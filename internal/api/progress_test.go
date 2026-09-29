package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"canvaslms-gui/internal/models"
	"github.com/stretchr/testify/require"
)

var testPollPolicy = pollPolicy{time.Millisecond, 4 * time.Millisecond, time.Second, 3}

func collectProgress(t *testing.T, updates <-chan models.BatchProgress) []models.BatchProgress {
	t.Helper()
	var out []models.BatchProgress
	deadline := time.After(2 * time.Second)
	for {
		select {
		case p, ok := <-updates:
			if !ok {
				return out
			}
			out = append(out, p)
		case <-deadline:
			t.Fatal("poller failed to terminate")
			return nil
		}
	}
}

func TestPollStatusAndTransientNetworkRecovery(t *testing.T) {
	calls := 0
	updates := pollProgress(context.Background(), func(context.Context) (models.BatchProgress, error) {
		calls++
		switch calls {
		case 1:
			return models.BatchProgress{WorkflowState: "queued", Completion: 100}, nil
		case 2:
			return models.BatchProgress{}, fmt.Errorf("temporary network failure")
		case 3:
			return models.BatchProgress{WorkflowState: "running", Completion: 0}, nil
		default:
			return models.BatchProgress{WorkflowState: "completed", Results: json.RawMessage(`{"errors":{"1001":"rejected"}}`)}, nil
		}
	}, testPollPolicy)
	out := collectProgress(t, updates)
	require.Len(t, out, 4)
	require.Equal(t, "queued", out[1].WorkflowState, "network errors must not masquerade as a Canvas job failure")
	require.Contains(t, out[1].PollingError, "retrying")
	require.Equal(t, "running", out[2].WorkflowState)
	require.Equal(t, "completed", out[3].WorkflowState)
	require.JSONEq(t, `{"errors":{"1001":"rejected"}}`, string(out[3].Results))
}

func TestPollBoundsNetworkFailures(t *testing.T) {
	calls := 0
	out := collectProgress(t, pollProgress(context.Background(), func(context.Context) (models.BatchProgress, error) {
		calls++
		return models.BatchProgress{}, fmt.Errorf("network unavailable")
	}, testPollPolicy))
	require.Equal(t, 3, calls)
	require.True(t, out[len(out)-1].PollingStopped)
	require.Contains(t, out[len(out)-1].Message, "may still be processing")
}

func TestPollPermanentAuthError(t *testing.T) {
	calls := 0
	out := collectProgress(t, pollProgress(context.Background(), func(context.Context) (models.BatchProgress, error) {
		calls++
		return models.BatchProgress{}, &AuthError{Message: "token rejected by Canvas"}
	}, testPollPolicy))
	require.Equal(t, 1, calls)
	require.True(t, out[0].PollingStopped)
}

func TestPollTimeoutCancelsRequest(t *testing.T) {
	policy := testPollPolicy
	policy.timeout = 20 * time.Millisecond
	out := collectProgress(t, pollProgress(context.Background(), func(ctx context.Context) (models.BatchProgress, error) {
		<-ctx.Done()
		return models.BatchProgress{}, ctx.Err()
	}, policy))
	require.Len(t, out, 1)
	require.True(t, out[0].PollingStopped)
	require.Contains(t, out[0].Message, "timed out")
}

func TestPollCancellationWithUndrainedUpdates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	updates := pollProgress(ctx, func(context.Context) (models.BatchProgress, error) {
		return models.BatchProgress{WorkflowState: "running"}, nil
	}, testPollPolicy)
	// Let the buffer fill: cancellation must also unblock an in-flight send.
	time.Sleep(15 * time.Millisecond)
	cancel()
	out := collectProgress(t, updates)
	for _, p := range out {
		require.NotEqual(t, "completed", p.WorkflowState)
	}
}

func TestPollCanvasFailedAndUnknownState(t *testing.T) {
	for _, state := range []string{"failed", "unexpected"} {
		out := collectProgress(t, pollProgress(context.Background(), func(context.Context) (models.BatchProgress, error) {
			return models.BatchProgress{WorkflowState: state, Message: "rejected"}, nil
		}, testPollPolicy))
		require.Len(t, out, 1)
		if state == "failed" {
			require.False(t, out[0].PollingStopped)
			require.Equal(t, "rejected", out[0].Message)
		} else {
			require.True(t, out[0].PollingStopped)
		}
	}
}

func TestProgressHTTPAndOriginValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/progress/42" {
			fmt.Fprint(w, `{"workflow_state":"completed","results":{"errors":{"1001":"rejected"}}}`)
			return
		}
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()
	client, err := NewCanvasClientWithToken(server.URL, "test-token")
	require.NoError(t, err)
	updates, err := client.PollBatchProgress(context.Background(), server.URL+"/api/v1/progress/42")
	require.NoError(t, err)
	out := collectProgress(t, updates)
	require.Len(t, out, 1)
	require.JSONEq(t, `{"errors":{"1001":"rejected"}}`, string(out[0].Results))
	_, err = client.PollBatchProgress(context.Background(), "https://foreign.example/progress/42")
	require.ErrorContains(t, err, "unexpected origin")
	_, err = client.BatchSubmitGrades(context.Background(), 1, 101, []models.GradeEntry{{StudentID: 1001, Grade: "95"}})
	require.ErrorContains(t, err, "no progress URL or valid job ID")
}
