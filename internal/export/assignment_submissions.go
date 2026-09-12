package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// NormalizeStudentName converts a student's display name into a filesystem-safe
// folder name:
//
//  1. Decompose Unicode to NFD and strip combining (accent) marks so that
//     "é" → "e", "ñ" → "n", etc.
//  2. Title-case each whitespace-separated word.
//  3. Replace every space (or run of spaces) with a single underscore.
//
// Example: "José María García" → "Jose_Maria_Garcia"
func NormalizeStudentName(name string) string {
	// Strip accents: NFD decomposition + remove non-spacing marks (Mn category).
	t := transform.Chain(
		norm.NFD,
		transform.RemoveFunc(func(r rune) bool {
			return unicode.Is(unicode.Mn, r)
		}),
		norm.NFC,
	)
	stripped, _, err := transform.String(t, name)
	if err != nil {
		stripped = name
	}

	// Title-case each word (first letter upper, rest lower).
	words := strings.Fields(stripped)
	for i, w := range words {
		if len(w) == 0 {
			continue
		}
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		for j := 1; j < len(runes); j++ {
			runes[j] = unicode.ToLower(runes[j])
		}
		words[i] = string(runes)
	}

	return strings.Join(words, "_")
}

// ExportAssignmentSubmissions downloads every file attachment from the
// submissions of a given assignment.  For each student that has at least one
// attached file a sub-directory named after the normalized student name is
// created inside dirPath and all attachments are downloaded into it.
//
// Progress is reported via the supplied ProgressFunc (may be nil).
// Returns a human-readable summary string on success.
func (e *canvasExporter) ExportAssignmentSubmissions(courseID, assignmentID int, dirPath string, progress ProgressFunc) (string, error) {
	// Fetch submissions (includes attachments).
	submissions, err := e.client.GetAssignmentSubmissionsWithAttachments(e.ctx, courseID, assignmentID)
	if err != nil {
		return "", fmt.Errorf("get submissions: %w", err)
	}

	// Build student map: id → User.
	students, err := e.client.GetStudentsForCourse(e.ctx, courseID)
	if err != nil {
		return "", fmt.Errorf("get students: %w", err)
	}
	studentMap := make(map[int]string, len(students))
	for _, s := range students {
		studentMap[s.ID] = s.Name
	}

	// Only process submissions that actually have file attachments.
	type work struct {
		folderName string
		files      []struct{ url, filename string }
	}
	var jobs []work
	for _, sub := range submissions {
		if len(sub.Attachments) == 0 {
			continue
		}
		name := studentMap[sub.UserID]
		if name == "" {
			name = fmt.Sprintf("User_%d", sub.UserID)
		}
		j := work{folderName: NormalizeStudentName(name)}
		for _, att := range sub.Attachments {
			j.files = append(j.files, struct{ url, filename string }{att.URL, att.Filename})
		}
		jobs = append(jobs, j)
	}

	if len(jobs) == 0 {
		return "No file submissions found for this assignment.", nil
	}

	downloaded := 0
	for i, job := range jobs {
		studentDir := filepath.Join(dirPath, job.folderName)
		if err := os.MkdirAll(studentDir, 0o755); err != nil {
			return "", fmt.Errorf("create folder %q: %w", job.folderName, err)
		}

		for _, f := range job.files {
			localPath := filepath.Join(studentDir, f.filename)
			if err := e.client.DownloadFile(e.ctx, f.url, localPath); err != nil {
				return "", fmt.Errorf("download %q for %s: %w", f.filename, job.folderName, err)
			}
		}

		downloaded++
		if progress != nil {
			progress(i+1, len(jobs), job.folderName)
		}
	}

	return fmt.Sprintf("%d student submission(s) saved to %s", downloaded, dirPath), nil
}
