package main

import (
	"dacapo/backend/app"
	"dacapo/backend/model"
	"dacapo/backend/router"
	"dacapo/backend/utils"
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {
	utils.InitLogger()
	defer utils.Logger.Sync()

	model.InitDB()

	r := router.SetupRouter()
	go r.Run(":48596")

	// Check if the symlink is valid, if not, create it
	paths, err := model.GetConfigPaths()
	if err == nil {
		for _, path := range paths {
			utils.CheckLink(path[0], path[1])
		}
	}

	// Create an instance of the app structure
	app := app.NewApp()

	// Create application with options
	err = wails.Run(&options.App{
		Title:             "DaCapo",
		Width:             1200,
		Height:            800,
		MinWidth:          750,
		MinHeight:         500,
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: false,
		BackgroundColour:  &options.RGBA{R: 255, G: 255, B: 255, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Menu:             nil,
		Logger:           utils.WailsLogger,
		LogLevel:         logger.DEBUG,
		OnStartup:        app.Startup,
		OnDomReady:       app.DomReady,
		OnBeforeClose:    app.BeforeClose,
		OnShutdown:       app.Shutdown,
		WindowStartState: options.Normal,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "95dcd591-3f9a-46d0-ac94-90b1ab2dce5f",
			OnSecondInstanceLaunch: app.OnSecondInstanceLaunch,
		},
		Bind: []interface{}{
			app,
		},
		// Windows platform specific options
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			// DisableFramelessWindowDecorations: false,
			WebviewUserDataPath: "",
			ZoomFactor:          1.0,
		},
		// Mac platform specific options
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
				HideToolbarSeparator:       true,
			},
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "DaCapo",
				Message: "",
				Icon:    icon,
			},
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}
