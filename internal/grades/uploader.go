package grades

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"canvaslms-gui/internal/api"
	apperrors "canvaslms-gui/internal/errors"
	"canvaslms-gui/internal/models"
)

// EventEmitter sends progress events to the frontend.
type EventEmitter interface {
	Emit(event string, data map[string]any)
}

// noopEmitter discards events (used when progress reporting is not needed).
type noopEmitter struct{}

func (noopEmitter) Emit(string, map[string]any) {}

// Uploader orchestrates the grade upload workflow.
type Uploader struct {
	client           api.CanvasClient
	emitter          EventEmitter
	ConfirmUnmatched func(context.Context, []StudentIssue) error
	Prepare          func(*LoadResult) error
	Cache            *UploadCache
	Submitted        bool // includes ambiguous submission-network failures; never auto-resubmit
}

// NewUploader creates a new Uploader. If emitter is nil, events are discarded.
func NewUploader(client api.CanvasClient, emitter EventEmitter) *Uploader {
	if emitter == nil {
		emitter = noopEmitter{}
	}
	return &Uploader{client: client, emitter: emitter}
}

// uploadedFile tracks metadata for a file uploaded to Canvas.
type uploadedFile struct {
	Column      string // pdf_exam_file1, pdf_exam_file2, or pdf_eval_file
	FileID      int
	Name        string
	URL         string
	DownloadURL string
	PublicURL   string
}

// gradeRow holds all data for one student during the upload flow.
type gradeRow struct {
	StudentID string
	Grade     float64
	Comment   string
	Files     []uploadedFile
}

// UploadGrades resolves/filters students and prepares all matched files before
// any Canvas writes, then uploads files concurrently, submits grades and monitors
// the status-only Canvas job. The caller decides whether unmatched rows may be
// skipped and reports errors with the appropriate retry policy.
func (u *Uploader) UploadGrades(ctx context.Context, courseID, assignmentID int, assignmentName string, result *LoadResult) error {
	if result == nil || len(result.Students) == 0 {
		return &apperrors.ValidationError{Field: "csv", Message: "no students to upload"}
	}

	// Resolve the roster before any file validation/conversion or Canvas writes.
	students, err := u.client.GetStudentsForCourse(ctx, courseID)
	if err != nil {
		return fmt.Errorf("get course roster: %w", err)
	}
	studentMap := make(map[string]string, len(students)) // ID → name
	for _, s := range students {
		studentMap[fmt.Sprintf("%d", s.ID)] = s.Name
	}

	matched, skipped := MatchStudents(result.Students, students)
	if len(skipped) > 0 {
		if u.ConfirmUnmatched == nil {
			return &apperrors.ValidationError{Field: "student_id", Message: "students not found in course roster"}
		}
		if err := u.ConfirmUnmatched(ctx, skipped); err != nil {
			return err
		}
	}
	result = &LoadResult{Students: matched, HasMD: result.HasMD}
	summary := map[string]any{"total": len(matched), "skipped": skipped, "failed": []StudentIssue{}}
	u.emitter.Emit("upload:summary", summary)
	if len(matched) == 0 {
		u.emitter.Emit("upload:done", summary)
		return nil
	}
	if u.Prepare != nil {
		if err := u.Prepare(result); err != nil {
			return err
		}
	}
	if err := NewLoader("", "").CheckFiles(result); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if u.Cache == nil {
		u.Cache = NewUploadCache()
	}

	folderName := generateFeedbackFolderName(assignmentName, assignmentID)
	u.emitter.Emit("upload:status", map[string]any{"message": "Creating feedback folder: " + folderName})
	folder, err := u.client.EnsureCourseFolder(ctx, courseID, folderName)
	if err != nil {
		return fmt.Errorf("create feedback folder: %w", err)
	}

	rows := make([]gradeRow, len(result.Students))
	for i, sg := range result.Students {
		rows[i] = gradeRow{
			StudentID: sg.StudentID,
			Grade:     sg.Grade,
			Comment:   sg.Comment,
		}
	}

	// Count total files across all students before starting
	totalFiles := 0
	for _, sg := range result.Students {
		if sg.PDFExamFile1 != "" {
			totalFiles++
		}
		if sg.PDFExamFile2 != "" {
			totalFiles++
		}
		if sg.PDFEvalFile != "" {
			totalFiles++
		}
	}

	// Step 3: upload files concurrently (Phase 1)
	u.emitter.Emit("upload:status", map[string]any{"message": "Uploading files..."})

	// Announce total so the frontend can initialize the progress bar immediately
	u.emitter.Emit("upload:file_start", map[string]any{
		"total":    totalFiles,
		"students": len(result.Students),
	})

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(5)

	var doneFiles int64    // atomic counter — goroutines run concurrently
	var filesMu sync.Mutex // multiple files can belong to the same student
	var issues []FileIssue

	for i := range result.Students {
		i := i
		sg := &result.Students[i]
		row := &rows[i]
		studentName := studentMap[sg.StudentID] // name is now in scope

		filePaths := map[string]string{
			"pdf_exam_file1": sg.PDFExamFile1,
			"pdf_exam_file2": sg.PDFExamFile2,
			"pdf_eval_file":  sg.PDFEvalFile,
		}

		for col, path := range filePaths {
			if path == "" {
				continue
			}

			g.Go(func() error {
				if err := gctx.Err(); err != nil {
					return err
				}
				info, err := u.Cache.Upload(gctx, u.client, courseID, assignmentID, folder.ID, path)
				if err != nil {
					if gctx.Err() != nil {
						return gctx.Err()
					}
					filesMu.Lock()
					issues = append(issues, FileIssue{sg.StudentID, path, "upload error: " + err.Error()})
					filesMu.Unlock()
					return nil // collect all failures; don't cancel other file uploads
				}
				filesMu.Lock()
				row.Files = append(row.Files, uploadedFile{
					Column:      col,
					FileID:      info.ID,
					Name:        info.Name,
					URL:         info.URL,
					DownloadURL: info.DownloadURL,
					PublicURL:   info.PublicURL,
				})
				filesMu.Unlock()
				done := atomic.AddInt64(&doneFiles, 1) // thread-safe increment
				u.emitter.Emit("upload:file_progress", map[string]any{
					"student": studentName, // name instead of ID
					"file":    filepath.Base(path),
					"done":    done,
					"total":   int64(totalFiles),
				})
				return nil
			})
		}
	}

	if err := g.Wait(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(issues) > 0 {
		return &FileProblems{Issues: issues}
	}

	// Step 4: build grade entries and batch submit (Phase 2)
	u.emitter.Emit("upload:status", map[string]any{"message": "Submitting grades..."})

	entries := make([]models.GradeEntry, len(rows))
	for i, row := range rows {
		entries[i] = models.GradeEntry{
			StudentID: atoi(row.StudentID),
			Grade:     formatGrade(row.Grade),
			Comment:   buildComment(row.Comment, row.Files),
		}
	}

	started := time.Now()
	u.emitter.Emit("upload:batch_progress", map[string]any{"state": "submitting", "elapsed_seconds": int64(0)})
	u.Submitted = true
	pollCtx, cancelPoll := context.WithTimeout(ctx, 16*time.Minute)
	defer cancelPoll()
	progressURL, err := u.client.BatchSubmitGrades(pollCtx, courseID, assignmentID, entries)
	if err != nil {
		return fmt.Errorf("batch submit grades: %w", err)
	}

	// Step 5: poll progress (Phase 3)
	u.emitter.Emit("upload:status", map[string]any{"message": "Waiting for Canvas to process grades..."})

	u.emitter.Emit("upload:batch_progress", map[string]any{"state": "queued", "elapsed_seconds": int64(time.Since(started).Seconds()), "progress_url": progressURL})
	updates, err := u.client.PollBatchProgress(pollCtx, progressURL)
	if err != nil {
		return fmt.Errorf("poll progress: %w", err)
	}

	for {
		var p models.BatchProgress
		select {
		case <-pollCtx.Done():
			return fmt.Errorf("grade monitoring stopped: %w", pollCtx.Err())
		case update, ok := <-updates:
			if !ok {
				if err := pollCtx.Err(); err != nil {
					return fmt.Errorf("grade monitoring stopped: %w", err)
				}
				return fmt.Errorf("Canvas progress stream ended without a completed or failed job")
			}
			p = update
		}
		u.emitter.Emit("upload:batch_progress", map[string]any{
			"state":           p.WorkflowState,
			"elapsed_seconds": int64(time.Since(started).Seconds()),
			"progress_url":    progressURL,
			"polling_error":   p.PollingError,
		})
		if p.PollingStopped {
			return fmt.Errorf("%s", p.Message)
		}
		switch p.WorkflowState {
		case "completed":
			u.emitter.Emit("upload:done", map[string]any{"total": len(entries), "skipped": skipped, "failed": JobIssues(p), "message": p.Message})
			return nil
		case "failed":
			u.emitter.Emit("upload:summary", map[string]any{"total": len(entries), "skipped": skipped, "failed": JobIssues(p), "message": p.Message})
			return fmt.Errorf("grade upload failed: %s", p.Message)
		case "queued", "running":
			// The completion number is deliberately ignored.
		default:
			return fmt.Errorf("Canvas returned an unknown job state: %s", p.WorkflowState)
		}
	}
}

// --- helpers ---

func generateFeedbackFolderName(assignmentName string, assignmentID int) string {
	now := time.Now()
	today := now.Format("2006-01-02")

	if assignmentName != "" {
		clean := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' || r == '-' || r == '_' {
				return r
			}
			return -1
		}, assignmentName)
		clean = strings.ReplaceAll(strings.TrimSpace(clean), " ", "_")
		if len(clean) > 30 {
			clean = clean[:30]
		}
		return fmt.Sprintf("Grade_Feedback/%s_%s", today, clean)
	}

	if assignmentID > 0 {
		return fmt.Sprintf("Grade_Feedback/%s_Assignment_%d", today, assignmentID)
	}

	timestamp := now.Format("2006-01-02_1504")
	return fmt.Sprintf("Grade_Feedback/%s_Manual_Upload", timestamp)
}

func formatGrade(g float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", g), "0"), ".")
}

func atoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func buildComment(textComment string, files []uploadedFile) string {
	var buf bytes.Buffer
	if err := commentTmpl.Execute(&buf, commentData{
		Comment: textComment,
		Files:   files,
	}); err != nil {
		// Fallback to plain text if template fails — never lose grade data.
		return textComment
	}
	return buf.String()
}

// Compile-time check: models.GradeEntry is compatible
var _ = models.GradeEntry{}
