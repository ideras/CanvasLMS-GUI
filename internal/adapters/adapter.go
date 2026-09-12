package adapters

import (
	"canvaslms-gui/internal/grades"
	"context"
	markdown "github.com/ideras/md-to-pdf"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// wailsEmitter adapts Wails runtime events to the grades.EventEmitter interface.
type wailsEmitter struct {
	ctx context.Context
}

func (e *wailsEmitter) Emit(event string, data map[string]any) {
	runtime.EventsEmit(e.ctx, event, data)
}

// mdConverter adapts the markdown converter to the grades.MarkdownConverter interface.
type mdConverter struct{}

func (c *mdConverter) ConvertFile(inputPath, outputPath string) error {
	return markdown.ConvertFile(inputPath, outputPath)
}

func NewWailsEmitter(ctx context.Context) grades.EventEmitter {
	return &wailsEmitter{ctx: ctx}
}

func NewMarkdownConverter() grades.MarkdownConverter {
	return &mdConverter{}
}
