package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"canvaslms-gui/internal/api"
	"canvaslms-gui/internal/models"
	"github.com/stretchr/testify/require"
)

type waitingRosterClient struct{ api.CanvasClient }

func (*waitingRosterClient) GetStudentsForCourse(ctx context.Context, _ int) ([]models.User, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestUploadRejectsConcurrentJobAndShutdownCancels(t *testing.T) {
	path := filepath.Join(t.TempDir(), "grades.csv")
	require.NoError(t, os.WriteFile(path, []byte("student_id,grade\n1001,95\n"), 0600))
	a := NewApp()
	a.ctx = context.Background()
	a.client = &waitingRosterClient{}
	require.NoError(t, a.UploadGrades(1, 101, path))
	require.True(t, a.uploadRunning())
	require.ErrorContains(t, a.UploadGrades(1, 101, path), "already running")
	done := a.uploadDone
	require.NoError(t, a.ServiceShutdown())
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not terminate the upload worker")
	}
	require.False(t, a.uploadRunning())
}
