package uploader

import (
	"context"
	"sync"
	"testing"
	"time"
)

// recordingEmitter captures event names emitted by the upload workflow,
// proving emissions go through the injected emitter (no Wails coupling).
type recordingEmitter struct {
	mu     sync.Mutex
	events []string
}

func (r *recordingEmitter) Emit(event string, data map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *recordingEmitter) names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.events))
	copy(out, r.events)
	return out
}

// TestUploadGradesReportsThroughInjectedEmitter verifies that a load failure
// is reported as upload:error through the injected emitter, and that the
// done channel is closed when the background work finishes.
func TestUploadGradesReportsThroughInjectedEmitter(t *testing.T) {
	em := &recordingEmitter{}
	cancel, done := UploadGrades(context.Background(), nil, em, 1, 2, "/nonexistent/grades.csv")

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("upload goroutine did not finish after load failure")
	}
	cancel() // must be safe to call after completion

	names := em.names()
	if len(names) == 0 {
		t.Fatal("expected at least one event, got none")
	}
	if names[0] != "upload:error" {
		t.Fatalf("expected first event upload:error, got %q", names[0])
	}
}