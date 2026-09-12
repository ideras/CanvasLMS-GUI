package uploader

import (
	"canvaslms-gui/internal/adapters"
	"canvaslms-gui/internal/api"
	"canvaslms-gui/internal/grades"
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func UploadGrades(appCtx context.Context, client api.CanvasClient, courseID int, assignmentID int, csvPath string) context.CancelFunc {
	ctx, cancel := context.WithCancel(appCtx)

	emitter := adapters.NewWailsEmitter(ctx)

	go func() {
		defer cancel()

		// Load CSV
		loader := grades.NewLoader(csvPath, "")
		result, err := loader.Load()
		if err != nil {
			runtime.EventsEmit(ctx, "upload:error", map[string]any{"error": err.Error()})
			return
		}

		// Convert MD files if present
		if result.HasMD {
			runtime.EventsEmit(ctx, "upload:status", map[string]any{"message": "Converting Markdown files to PDF..."})
			if err := loader.ConvertMarkdownFiles(result, adapters.NewMarkdownConverter()); err != nil {
				runtime.EventsEmit(ctx, "upload:error", map[string]any{"error": err.Error()})
				return
			}
		}

		// Run upload
		uploader := grades.NewUploader(client, emitter)
		if err := uploader.UploadGrades(ctx, courseID, assignmentID, "", result); err != nil {
			runtime.EventsEmit(ctx, "upload:error", map[string]any{"error": err.Error()})
			return
		}
	}()

	return cancel
}
