package adapters

import (
	"canvaslms-gui/internal/grades"

	"github.com/wailsapp/wails/v3/pkg/application"

	markdown "github.com/ideras/md-to-pdf"
)

// wailsEmitter forwards event emissions to the Wails v3 application event
// bus, preserving the v2 event contract (name + map payload).
type wailsEmitter struct{}

func (e *wailsEmitter) Emit(event string, data map[string]any) {
	if app := application.Get(); app != nil {
		app.Event.Emit(event, data)
	}
}

// mdConverter adapts the markdown converter to the grades.MarkdownConverter interface.
type mdConverter struct{}

func (c *mdConverter) ConvertFile(inputPath, outputPath string) error {
	return markdown.ConvertFile(inputPath, outputPath)
}

// NewWailsEmitter returns a grades.EventEmitter backed by the Wails v3
// application event bus. Emissions are dropped when the application has not
// been created (e.g. unit tests without a desktop runtime).
func NewWailsEmitter() grades.EventEmitter {
	return &wailsEmitter{}
}

func NewMarkdownConverter() grades.MarkdownConverter {
	return &mdConverter{}
}