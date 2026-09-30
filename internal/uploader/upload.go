package uploader

import (
	"canvaslms-gui/internal/adapters"
	"canvaslms-gui/internal/api"
	"canvaslms-gui/internal/grades"
	"context"
	"errors"
)

// UploadGrades runs the full grade upload workflow in the background. All
// progress/error reporting goes through the provided emitter so this package
// stays free of transport (Wails) coupling. Returns the cancel function and a
// channel closed when the background work finishes (for bounded shutdown waits).
func UploadGrades(appCtx context.Context, client api.CanvasClient, emitter grades.EventEmitter, courseID int, assignmentID int, csvPath string) (context.CancelFunc, <-chan struct{}) {
	return Start(appCtx, client, emitter, courseID, assignmentID, csvPath, nil, nil)
}

// Start accepts user decisions without coupling the worker to the desktop.
func Start(appCtx context.Context, client api.CanvasClient, emitter grades.EventEmitter, courseID, assignmentID int, csvPath string, decisions <-chan string, uploadCache *grades.UploadCache) (context.CancelFunc, <-chan struct{}) {
	ctx, cancel := context.WithCancel(appCtx)
	done := make(chan struct{})

	go func() {
		defer cancel()
		defer close(done)

		// Load CSV
		loader := grades.NewLoader(csvPath, "")
		result, err := loader.Parse()
		if err != nil {
			reportError(emitter, err, true)
			return
		}

		// File preparation runs only after roster confirmation.
		uploader := grades.NewUploader(client, emitter)
		uploader.Cache = uploadCache
		if decisions != nil {
			uploader.ConfirmUnmatched = func(ctx context.Context, students []grades.StudentIssue) error {
				return grades.ConfirmStudents(ctx, emitter, decisions, students)
			}
		}
		uploader.Prepare = func(result *grades.LoadResult) error {
			if err := loader.CheckFiles(result); err != nil {
				return err
			}
			if result.HasMD {
				emitter.Emit("upload:status", map[string]any{"message": "Converting Markdown files to PDF..."})
				converter := adapters.NewMarkdownConverter()
				if uploadCache != nil {
					converter = uploadCache.Converter(converter)
				}
				return loader.ConvertMarkdownFiles(result, converter)
			}
			return nil
		}
		if err := uploader.UploadGrades(ctx, courseID, assignmentID, "", result); err != nil {
			reportError(emitter, err, !uploader.Submitted)
			return
		}
	}()

	return cancel, done
}

func reportError(emitter grades.EventEmitter, err error, retryable bool) {
	if errors.Is(err, context.Canceled) {
		emitter.Emit("upload:cancelled", map[string]any{})
		return
	}
	data := map[string]any{"error": err.Error(), "retryable": retryable}
	var problems *grades.FileProblems
	if errors.As(err, &problems) {
		data["files"] = problems.Issues
	}
	if !retryable {
		data["error"] = err.Error() + ". Grades may already have been applied; check Canvas before starting another upload."
	}
	emitter.Emit("upload:error", data)
}
