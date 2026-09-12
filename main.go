package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	wailsApp := application.New(application.Options{
		Name:        "CanvasLMS GUI",
		Description: "Desktop GUI for Canvas LMS management",
		Services: []application.Service{
			application.NewService(app),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Create the main window. The returned handle is handed to the service
	// (kept private) so desktop operations such as dialogs attach to it.
	mainWindow := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "CanvasLMS GUI",
		Width:            1100,
		Height:           750,
		MinWidth:         800,
		MinHeight:        600,
		BackgroundColour: application.NewRGBA(214, 234, 248, 255),
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBar{
				AppearsTransparent:   true,
				HideTitle:            true,
				FullSizeContent:      true,
				UseToolbar:           true,
				HideToolbarSeparator: true,
			},
		},
		URL: "/",
	})

	app.setDesktop(wailsApp, mainWindow)

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}