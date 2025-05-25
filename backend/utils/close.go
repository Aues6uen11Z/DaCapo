package utils

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	appContext context.Context
	contextMu  sync.RWMutex
)

func SetAppContext(ctx context.Context) {
	contextMu.Lock()
	defer contextMu.Unlock()
	appContext = ctx
}

func GetAppContext() context.Context {
	contextMu.RLock()
	defer contextMu.RUnlock()
	return appContext
}

// CloseApp closes the application
func CloseApp() {
	Logger.Info("Application shutdown...")

	ctx := GetAppContext()
	if ctx == nil {
		Logger.Warn("App context not available, using force close")
		os.Exit(0)
	}

	// Give the application some time to complete current operations
	time.Sleep(3 * time.Second)

	// Use Wails runtime to quit
	wailsRuntime.Quit(ctx)
}

// Hibernate puts Windows system into hibernation mode
func Hibernate() {
	if runtime.GOOS != "windows" {
		Logger.Warn("Hibernate is only supported on Windows")
	}

	Logger.Info("Initiating system hibernation...")

	// Use Windows shutdown command to hibernate
	cmd := exec.Command("shutdown", "/h", "/f")
	err := cmd.Run()
	if err != nil {
		Logger.Errorf("Failed to hibernate system: %v", err)
	}
}

// Shutdown shuts down the system
func Shutdown() {
	Logger.Info("Initiating system shutdown...")

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Windows: shutdown after 10 seconds, can be cancelled
		cmd = exec.Command("shutdown", "/s", "/t", "10", "/f")
	case "linux":
		cmd = exec.Command("sudo", "shutdown", "-h", "+1")
	case "darwin":
		cmd = exec.Command("sudo", "shutdown", "-h", "+1")
	default:
		Logger.Warn("Shutdown is not supported on this operating system")
	}

	err := cmd.Run()
	if err != nil {
		Logger.Errorf("Failed to shutdown system: %v", err)
	}
}
