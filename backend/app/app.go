package app

import (
	"context"
	"dacapo/backend/utils"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Define version constant
const Version = "1.1.0"

// var wailsContext *context.Context

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// GetVersion returns the application version number
func (a *App) GetVersion() string {
	return Version
}

// startup is called at application startup
func (a *App) Startup(ctx context.Context) {
	utils.Logger.Info("Application is starting up")
	a.ctx = ctx
}

// OnSecondInstanceLaunch handles when a second instance of the app is launched
func (a *App) OnSecondInstanceLaunch(secondInstanceData options.SecondInstanceData) {
	secondInstanceArgs := secondInstanceData.Args

	utils.Logger.Infoln("User opened second instance", strings.Join(secondInstanceData.Args, ","))
	utils.Logger.Infoln("User opened second from", secondInstanceData.WorkingDirectory)
	runtime.WindowUnminimise(a.ctx)
	runtime.Show(a.ctx)
	go runtime.EventsEmit(a.ctx, "launchArgs", secondInstanceArgs)
}

// domReady is called after front-end resources have been loaded
func (a App) DomReady(ctx context.Context) {
	// Add your action here
}

// beforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	return false
}

// shutdown is called at application termination
func (a *App) Shutdown(ctx context.Context) {
	// Perform your teardown here
}

// SelectDir opens a directory selection dialog and returns the selected path
func (a *App) SelectDir() string {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		CanCreateDirectories: true,
	})
	if err != nil {
		utils.Logger.Error(err.Error())
	}
	return path
}

// SelectFile opens a file selection dialog and returns the selected path
func (a *App) SelectFile() string {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		CanCreateDirectories: true,
	})
	if err != nil {
		utils.Logger.Error(err.Error())
	}
	return path
}

// OpenFileExplorer opens the file explorer at the specified directory path
func (a *App) OpenFileExplorer(workDir, logPath string) {
	var absPath string
	if filepath.IsAbs(logPath) {
		absPath = logPath
	} else {
		absPath = filepath.Join(workDir, logPath)
	}

	cmd := exec.Command("explorer.exe", absPath)
	err := cmd.Start()
	if err != nil {
		utils.Logger.Error(err.Error())
	}
}
