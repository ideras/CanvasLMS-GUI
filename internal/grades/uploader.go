package grades

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
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
	client  api.CanvasClient
	emitter EventEmitter
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

// UploadGrades runs the full grade upload workflow:
//  1. Create feedback folder in Canvas
//  2. Upload all PDF files concurrently (Phase 1)
//  3. Batch-submit grades (Phase 2)
//  4. Poll Canvas progress until complete (Phase 3)
func (u *Uploader) UploadGrades(ctx context.Context, courseID, assignmentID int, assignmentName string, result *LoadResult) error {
	if result == nil || len(result.Students) == 0 {
		return &apperrors.ValidationError{Field: "csv", Message: "no students to upload"}
	}

	// Step 1: create feedback folder
	folderName := generateFeedbackFolderName(assignmentName, assignmentID)
	u.emitter.Emit("upload:status", map[string]any{"message": "Creating feedback folder: " + folderName})

	folder, err := u.client.EnsureCourseFolder(ctx, courseID, folderName)
	if err != nil {
		return fmt.Errorf("create feedback folder: %w", err)
	}

	// Step 2: validate student IDs against course roster
	students, err := u.client.GetStudentsForCourse(ctx, courseID)
	if err != nil {
		return fmt.Errorf("get course roster: %w", err)
	}
	studentMap := make(map[string]string, len(students)) // ID → name
	for _, s := range students {
		studentMap[fmt.Sprintf("%d", s.ID)] = s.Name
	}

	// Validate and build rows
	rows := make([]gradeRow, len(result.Students))
	for i, sg := range result.Students {
		if _, ok := studentMap[sg.StudentID]; !ok {
			return &apperrors.ValidationError{
				Field:   "student_id",
				Message: fmt.Sprintf("student ID %s not found in course roster", sg.StudentID),
			}
		}
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

	var doneFiles int64 // atomic counter — goroutines run concurrently

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
				info, err := u.client.UploadFileToCourse(gctx, path, courseID, folder.ID)
				if err != nil {
					return fmt.Errorf("upload %s for student %s: %w", col, sg.StudentID, err)
				}
				row.Files = append(row.Files, uploadedFile{
					Column:      col,
					FileID:      info.ID,
					Name:        info.Name,
					URL:         info.URL,
					DownloadURL: info.DownloadURL,
					PublicURL:   info.PublicURL,
				})
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
		u.emitter.Emit("upload:error", map[string]any{"error": err.Error()})
		return err
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

	progressURL, err := u.client.BatchSubmitGrades(ctx, courseID, assignmentID, entries)
	if err != nil {
		u.emitter.Emit("upload:error", map[string]any{"error": err.Error()})
		return fmt.Errorf("batch submit grades: %w", err)
	}

	// Step 5: poll progress (Phase 3)
	u.emitter.Emit("upload:status", map[string]any{"message": "Waiting for Canvas to process grades..."})

	updates, err := u.client.PollBatchProgress(ctx, progressURL)
	if err != nil {
		u.emitter.Emit("upload:error", map[string]any{"error": err.Error()})
		return fmt.Errorf("poll progress: %w", err)
	}

	for p := range updates {
		switch p.WorkflowState {
		case "completed":
			u.emitter.Emit("upload:done", map[string]any{"total": len(entries)})
			return nil
		case "failed":
			u.emitter.Emit("upload:error", map[string]any{"error": p.Message})
			return fmt.Errorf("grade upload failed: %s", p.Message)
		default: // "queued" or "running"
			u.emitter.Emit("upload:batch_progress", map[string]any{
				"completion": p.Completion,
				"state":      p.WorkflowState,
			})
		}
	}

	return nil
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
