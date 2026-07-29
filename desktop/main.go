package main

import (
	"embed"
	// Embed the IANA timezone database so calendar-invite times (TZID=…) resolve
	// even when the packaged app has no access to the system zoneinfo.
	_ "time/tzdata"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "GizTUI Desktop",
		Width:     1280,
		Height:    832,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 20, G: 22, B: 27, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title:   "GizTUI Desktop",
				Message: "A visual Gmail client powered by the GizTUI service layer.",
			},
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
