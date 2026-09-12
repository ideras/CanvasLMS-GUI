package adapters

import (
	"testing"

	"canvaslms-gui/internal/grades"
)

// TestWailsEmitterDropsWithoutApp verifies Emit is a safe no-op (no panic)
// when the Wails application has not been created, e.g. in unit tests or
// non-desktop contexts.
func TestWailsEmitterDropsWithoutApp(t *testing.T) {
	var emitter grades.EventEmitter = NewWailsEmitter()
	emitter.Emit("upload:status", map[string]any{"message": "hello"})
	emitter.Emit("upload:error", map[string]any{"error": "boom"})
}

// TestMarkdownConverterRejectsMissingInput checks the converter implements
// the grades.MarkdownConverter seam and fails cleanly on missing input.
func TestMarkdownConverterRejectsMissingInput(t *testing.T) {
	var c grades.MarkdownConverter = NewMarkdownConverter()
	if err := c.ConvertFile("/nonexistent/path/input.md", "/tmp/canvaslms-gui-test-out.pdf"); err == nil {
		t.Fatal("expected error converting nonexistent input file")
	}
}