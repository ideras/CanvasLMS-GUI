package uploader

import (
	"canvaslms-gui/internal/adapters"
	"canvaslms-gui/internal/api"
	"canvaslms-gui/internal/grades"
	"context"
)

// UploadGrades runs the full grade upload workflow in the background. All
// progress/error reporting goes through the provided emitter so this package
// stays free of transport (Wails) coupling. Returns the cancel function and a
// channel closed when the background work finishes (for bounded shutdown waits).
func UploadGrades(appCtx context.Context, client api.CanvasClient, emitter grades.EventEmitter, courseID int, assignmentID int, csvPath string) (context.CancelFunc, <-chan struct{}) {
	ctx, cancel := context.WithCancel(appCtx)
	done := make(chan struct{})

	go func() {
		defer cancel()
		defer close(done)

		// Load CSV
		loader := grades.NewLoader(csvPath, "")
		result, err := loader.Load()
		if err != nil {
			emitter.Emit("upload:error", map[string]any{"error": err.Error()})
			return
		}

		// Convert MD files if present
		if result.HasMD {
			emitter.Emit("upload:status", map[string]any{"message": "Converting Markdown files to PDF..."})
			if err := loader.ConvertMarkdownFiles(result, adapters.NewMarkdownConverter()); err != nil {
				emitter.Emit("upload:error", map[string]any{"error": err.Error()})
				return
			}
		}

		// Run upload
		uploader := grades.NewUploader(client, emitter)
		if err := uploader.UploadGrades(ctx, courseID, assignmentID, "", result); err != nil {
			emitter.Emit("upload:error", map[string]any{"error": err.Error()})
			return
		}
	}()

	return cancel, done
}