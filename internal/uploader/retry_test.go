package uploader

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"canvaslms-gui/internal/api"
	"canvaslms-gui/internal/grades"
	"canvaslms-gui/internal/models"
	"github.com/stretchr/testify/require"
)

type retryClient struct {
	api.CanvasClient
	rosterCalls int
	entries     []models.GradeEntry
}

func (c *retryClient) GetStudentsForCourse(context.Context, int) ([]models.User, error) {
	c.rosterCalls++
	return []models.User{{ID: 1001}}, nil
}
func (c *retryClient) EnsureCourseFolder(context.Context, int, string) (*models.Folder, error) {
	return &models.Folder{ID: 1}, nil
}
func (c *retryClient) BatchSubmitGrades(_ context.Context, _, _ int, entries []models.GradeEntry) (string, error) {
	c.entries = entries
	return "url", nil
}
func (c *retryClient) PollBatchProgress(context.Context, string) (<-chan models.BatchProgress, error) {
	ch := make(chan models.BatchProgress, 1)
	ch <- models.BatchProgress{WorkflowState: "completed"}
	close(ch)
	return ch, nil
}

func TestRetryReadsCurrentCSVAndRefetchesRoster(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.csv")
	second := filepath.Join(dir, "second.csv")
	require.NoError(t, os.WriteFile(first, []byte("student_id,grade,pdf_eval_file\n1001,50,missing.pdf\n"), 0600))
	require.NoError(t, os.WriteFile(second, []byte("student_id,grade\n1001,95\n"), 0600))
	client := &retryClient{}
	cache := grades.NewUploadCache()
	run := func(path string) []string {
		emitter := &recordingEmitter{}
		cancel, done := Start(context.Background(), client, emitter, 1, 101, path, nil, cache)
		defer cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("worker timed out")
		}
		return emitter.names()
	}
	require.Contains(t, run(first), "upload:error")
	require.Empty(t, client.entries)
	require.Contains(t, run(second), "upload:done")
	require.Equal(t, 2, client.rosterCalls)
	require.Equal(t, "95", client.entries[0].Grade)
}
