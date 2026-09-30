package grades_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"canvaslms-gui/internal/grades"
	"canvaslms-gui/internal/models"
	"github.com/stretchr/testify/require"
)

type progressClient struct {
	*mockClient
	updates <-chan models.BatchProgress
}

func (c *progressClient) PollBatchProgress(context.Context, string) (<-chan models.BatchProgress, error) {
	return c.updates, nil
}

type payloadEmitter struct {
	spyEmitter
	payloads map[string]map[string]any
}

func (e *payloadEmitter) Emit(event string, data map[string]any) {
	e.spyEmitter.Emit(event, data)
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.payloads == nil {
		e.payloads = make(map[string]map[string]any)
	}
	e.payloads[event] = data
}

func TestClosedProgressIsNotSuccess(t *testing.T) {
	ch := make(chan models.BatchProgress)
	close(ch)
	client := &progressClient{newMockClient(), ch}
	emitter := &spyEmitter{}
	u := grades.NewUploader(client, emitter)
	err := u.UploadGrades(context.Background(), 1, 101, "", &grades.LoadResult{Students: []grades.StudentGrade{{StudentID: "1001"}}})
	require.ErrorContains(t, err, "without a completed or failed job")
	require.False(t, emitter.contains(t, "upload:done"))
}

func TestProgressWaitIsCancellable(t *testing.T) {
	client := &progressClient{newMockClient(), make(chan models.BatchProgress)}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := grades.NewUploader(client, nil).UploadGrades(ctx, 1, 101, "", &grades.LoadResult{Students: []grades.StudentGrade{{StudentID: "1001"}}})
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestCompletedProgressReportsPerStudentFailures(t *testing.T) {
	ch := make(chan models.BatchProgress, 1)
	ch <- models.BatchProgress{WorkflowState: "completed", Results: json.RawMessage(`{"errors":{"1001":"grade rejected"}}`)}
	close(ch)
	emitter := &payloadEmitter{}
	u := grades.NewUploader(&progressClient{newMockClient(), ch}, emitter)
	u.ConfirmUnmatched = func(context.Context, []grades.StudentIssue) error { return nil }
	err := u.UploadGrades(context.Background(), 1, 101, "", &grades.LoadResult{Students: []grades.StudentGrade{{StudentID: "1001"}, {StudentID: "9999"}}})
	require.NoError(t, err)
	require.Len(t, emitter.payloads["upload:done"]["skipped"], 1)
	require.Len(t, emitter.payloads["upload:done"]["failed"], 1)
	require.NotContains(t, emitter.payloads["upload:batch_progress"], "completion")
	require.Equal(t, "completed", emitter.payloads["upload:batch_progress"]["state"])
}
