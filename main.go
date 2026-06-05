package main

import (
	"embed"
	"log"

	"github.com/masterdatamate/masterdatamate/internal/host"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed dist
var embeddedDist embed.FS

func main() {
	workspace, err := host.ResolveWorkspace(".")
	if err != nil {
		log.Fatal(err)
	}
	handler, err := host.Handler(workspace, embeddedDist)
	if err != nil {
		log.Fatal(err)
	}
	if err := wails.Run(&options.App{
		Title:  "MasterDataMate",
		Width:  1280,
		Height: 860,
		AssetServer: &assetserver.Options{
			Handler: handler,
		},
	}); err != nil {
		log.Fatal(err)
	}
}
