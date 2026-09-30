package grades

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"canvaslms-gui/internal/api"
	"canvaslms-gui/internal/models"
)

// FileIssue preserves the student, path and reason for repair/retry dialogs.
type FileIssue struct {
	StudentID string `json:"student_id"`
	File      string `json:"file"`
	Reason    string `json:"reason"`
}

type FileProblems struct{ Issues []FileIssue }

func (e *FileProblems) Error() string {
	lines := make([]string, 0, len(e.Issues))
	for _, issue := range e.Issues {
		lines = append(lines, fmt.Sprintf("student %s — %s: %s", issue.StudentID, issue.File, issue.Reason))
	}
	return strings.Join(lines, "\n")
}

func fileReason(err error) string {
	switch {
	case os.IsNotExist(err):
		return "not found"
	case os.IsPermission(err):
		return "permission denied"
	default:
		return "cannot read: " + err.Error()
	}
}

// CheckReadable reads the complete regular file, detecting permission and I/O
// errors up front. All callers run on the worker, never the GUI thread.
func CheckReadable(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file")
	}
	if info.Size() > 50*1024*1024 {
		return fmt.Errorf("file exceeds Canvas client's 50 MB limit")
	}
	_, err = io.Copy(io.Discard, f)
	return err
}

// UploadCache remembers only acknowledged Canvas uploads within one workflow.
// Keys include the target, absolute local path and content hash. Per-key locks
// prevent duplicate uploads of a shared file without serializing unrelated files.
type UploadCache struct {
	mu          sync.Mutex
	files       map[string]models.FileInfo
	locks       map[string]*sync.Mutex
	conversions map[string]conversionRecord
}

func NewUploadCache() *UploadCache {
	return &UploadCache{files: make(map[string]models.FileInfo), locks: make(map[string]*sync.Mutex), conversions: make(map[string]conversionRecord)}
}

func (c *UploadCache) Upload(ctx context.Context, client api.CanvasClient, courseID, assignmentID, folderID int, path string) (*models.FileInfo, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(absolute)
	if err != nil {
		return nil, err
	}
	hash := sha256.New()
	_, err = io.Copy(hash, f)
	f.Close()
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%d/%d/%s/%x", courseID, assignmentID, absolute, hash.Sum(nil))
	c.mu.Lock()
	lock := c.locks[key]
	if lock == nil {
		lock = &sync.Mutex{}
		c.locks[key] = lock
	}
	c.mu.Unlock()
	lock.Lock()
	defer lock.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	info, ok := c.files[key]
	c.mu.Unlock()
	if ok {
		return &info, nil
	}
	uploaded, err := client.UploadFileToCourse(ctx, path, courseID, folderID)
	if err != nil {
		return nil, err
	}
	if uploaded == nil || uploaded.ID <= 0 {
		return nil, fmt.Errorf("Canvas returned no confirmed file ID")
	}
	c.mu.Lock()
	c.files[key] = *uploaded
	c.mu.Unlock()
	return uploaded, nil
}
