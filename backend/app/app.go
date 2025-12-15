package app

import (
	"context"
	"dacapo/backend/controller"
	"dacapo/backend/model"
	"dacapo/backend/service"
	"dacapo/backend/utils"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Define version constant
// This is a placeholder. The actual version is set during CI/CD build.
const Version = "0.0.0-dev"

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

	utils.SetAppVersion(Version)
	utils.SetAppContext(ctx)

	// Check if the symlink is valid, if not, create it
	paths, err := model.GetConfigPaths()
	if err == nil {
		for _, path := range paths {
			utils.CheckLink(path[0], path[1])
		}
	}

	// Start file watcher for instance configuration files
	fileWatcher := controller.GetFileWatcher()
	if fileWatcher != nil {
		if err := fileWatcher.Start(); err != nil {
			utils.Logger.Errorf("Failed to start file watcher: %v", err)
		}
	}

	// Check if the old executable file exists and delete it
	oldFilePath := ".DaCapo.exe.old"
	if _, err := os.Stat(oldFilePath); err == nil {
		err := os.Remove(oldFilePath)
		if err != nil {
			utils.Logger.Errorf("Failed to delete old executable file: %v", err)
			return
		}
		utils.Logger.Infof("Deleted old executable file: %s", oldFilePath)
	}
}

// OnSecondInstanceLaunch handles when a second instance of the app is launched
func (a *App) OnSecondInstanceLaunch(secondInstanceData options.SecondInstanceData) {
	secondInstanceArgs := secondInstanceData.Args

	utils.Logger.Infoln("User opened second instance from", secondInstanceData.WorkingDirectory)
	runtime.WindowUnminimise(a.ctx)
	runtime.Show(a.ctx)
	go runtime.EventsEmit(a.ctx, "launchArgs", secondInstanceArgs)
}

// domReady is called after front-end resources have been loaded
func (a App) DomReady(ctx context.Context) {
	// Load settings from settings file
	settings, err := model.LoadSettings()
	if err != nil {
		utils.Logger.Error("Failed to load settings:", err)
	}

	var instances []model.InstanceInfo
	if err := model.GetAllInstances(&instances); err != nil {
		utils.Logger.Error("Failed to get all instances:", err)
		return
	}

	c := cron.New()
	for _, instance := range instances {
		if instance.Ready && instance.CronExpr != "" {
			entryID, err := c.AddFunc(instance.CronExpr, func() {
				service.GetServiceManager().SchedulerService().StartOne(instance.Name)
			})
			if err != nil {
				utils.Logger.Errorf("[%s]: Failed to add cron job: %v", instance.Name, err)
			} else {
				entry := c.Entry(entryID)
				nextRun := entry.Schedule.Next(time.Now()).Format("2006-01-02 15:04:05")
				utils.Logger.Infof("[%s]: Cron job added: %s, next run at %s", instance.Name, instance.CronExpr, nextRun)
			}
		}
	}

	time.Sleep(3 * time.Second)
	scheduler := model.GetScheduler()
	// Handle runOnStartup setting
	if settings.RunOnStartup {
		utils.Logger.Info("Run on startup is enabled, starting scheduler")
		scheduler.AutoClose = true
		go service.GetServiceManager().SchedulerService().StartAll()
	} else if settings.SchedulerCron != "" {
		// If runOnStartup is false but schedulerCron is set, use cron scheduling
		scheduler.CronExpr = settings.SchedulerCron
		entryID, err := c.AddFunc(scheduler.CronExpr, func() {
			scheduler.AutoClose = true
			service.GetServiceManager().SchedulerService().StartAll()
		})
		if err != nil {
			utils.Logger.Errorf("Scheduler failed to add cron job: %v", err)
		} else {
			entry := c.Entry(entryID)
			nextRun := entry.Schedule.Next(time.Now()).Format("2006-01-02 15:04:05")
			utils.Logger.Infof("Scheduler cron job added: %s, next run at %s", scheduler.CronExpr, nextRun)
		}
	}

	// Auto close
	var closeFunc func()
	switch settings.AutoActionType {
	case "close_app":
		closeFunc = utils.CloseApp
	case "hibernate":
		closeFunc = utils.Hibernate
	case "shutdown":
		closeFunc = utils.Shutdown
	}

	if settings.AutoActionTrigger == "scheduled" && settings.AutoActionCron != "" && closeFunc != nil {
		entryID, err := c.AddFunc(settings.AutoActionCron, func() {
			if scheduler.AutoClose {
				closeFunc()
			}
		})
		if err != nil {
			utils.Logger.Errorf("Failed to add auto close cron job: %v", err)
		} else {
			entry := c.Entry(entryID)
			nextRun := entry.Schedule.Next(time.Now()).Format("2006-01-02 15:04:05")
			utils.Logger.Infof("Auto close cron job added: %s, next run at %s", settings.AutoActionCron, nextRun)
		}
	} else if settings.AutoActionTrigger == "scheduler_end" {
		scheduler.CloseFunc = closeFunc
	}

	c.Start()
}

// beforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	service.GetServiceManager().SchedulerService().StopAll()

	// Stop file watcher
	fileWatcher := controller.GetFileWatcher()
	if fileWatcher != nil && fileWatcher.IsRunning() {
		fileWatcher.Stop()
	}

	model.CloseDB()
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
