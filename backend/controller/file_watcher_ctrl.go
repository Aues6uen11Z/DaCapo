package controller

import (
	"dacapo/backend/model"
	"dacapo/backend/utils"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher manages file system monitoring for instance configuration files
type FileWatcher struct {
	watcher      *fsnotify.Watcher
	mu           sync.RWMutex
	isRunning    bool
	stopCh       chan struct{}
	ignoredMu    sync.RWMutex
	ignoredFiles map[string]time.Time // Track files to ignore with their ignore expiry time
}

var fileWatcher *FileWatcher

// GetFileWatcher returns the singleton file watcher instance
func GetFileWatcher() *FileWatcher {
	if fileWatcher == nil {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			utils.Logger.Errorf("Failed to create file watcher: %v", err)
			return nil
		}

		fileWatcher = &FileWatcher{
			watcher:      watcher,
			stopCh:       make(chan struct{}),
			ignoredFiles: make(map[string]time.Time),
		}
	}
	return fileWatcher
}

// Start begins monitoring instance configuration files
func (fw *FileWatcher) Start() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.isRunning {
		return nil
	}

	// Set up the callback to ignore programmatic file writes
	model.FileWriteCallback = fw.IgnoreFile

	// Ensure instances directory exists
	instancesDir := "instances"
	if err := os.MkdirAll(instancesDir, 0755); err != nil {
		utils.Logger.Errorf("Failed to create instances directory: %v", err)
		return err
	}

	// Watch the instances directory
	if err := fw.watcher.Add(instancesDir); err != nil {
		utils.Logger.Errorf("Failed to watch instances directory: %v", err)
		return err
	}

	fw.isRunning = true
	utils.Logger.Info("File watcher started, monitoring instances directory")

	// Start the monitoring goroutine
	go fw.watchLoop()

	return nil
}

// Stop stops the file watcher
func (fw *FileWatcher) Stop() {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if !fw.isRunning {
		return
	}

	// Clear the callback
	model.FileWriteCallback = nil

	close(fw.stopCh)
	fw.watcher.Close()
	fw.isRunning = false
	utils.Logger.Info("File watcher stopped")
}

// watchLoop processes file system events
func (fw *FileWatcher) watchLoop() {
	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			fw.handleFileEvent(event)

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			utils.Logger.Errorf("File watcher error: %v", err)

		case <-fw.stopCh:
			return
		}
	}
}

// handleFileEvent processes individual file system events
func (fw *FileWatcher) handleFileEvent(event fsnotify.Event) {
	// Only process file write events for JSON files in the instances directory
	if !strings.HasSuffix(event.Name, ".json") || event.Op&fsnotify.Write == 0 {
		return
	}

	// Check if this file should be ignored (programmatic write or recent external change)
	if fw.shouldIgnoreFile(event.Name) {
		return
	}

	// Extract instance name from filename
	filename := filepath.Base(event.Name)
	instanceName := strings.TrimSuffix(filename, ".json")

	// Mark this file to be ignored for a short time (prevent duplicate events)
	fw.ignoredMu.Lock()
	filePath := filepath.Join("instances", instanceName+".json")
	absolutePath, _ := filepath.Abs(filePath)
	fw.ignoredFiles[absolutePath] = time.Now().Add(500 * time.Millisecond) // Ignore for 500ms
	fw.ignoredMu.Unlock()

	utils.Logger.Infof("[%s]: external file modification detected", instanceName)
	fw.broadcastFileChange(instanceName, filename)
}

// IgnoreFile marks a file to be ignored for a short duration (to prevent detection of programmatic writes)
func (fw *FileWatcher) IgnoreFile(instanceName string) {
	fw.ignoredMu.Lock()
	defer fw.ignoredMu.Unlock()

	// Convert instance name to file path
	filePath := filepath.Join("instances", instanceName+".json")
	absolutePath, _ := filepath.Abs(filePath)

	// Ignore file changes for 2 seconds after programmatic write
	fw.ignoredFiles[absolutePath] = time.Now().Add(2 * time.Second)
}

// shouldIgnoreFile checks if a file should be ignored due to recent programmatic write
func (fw *FileWatcher) shouldIgnoreFile(filePath string) bool {
	fw.ignoredMu.Lock()
	defer fw.ignoredMu.Unlock()

	// Clean up expired ignore entries
	now := time.Now()
	for path, expiry := range fw.ignoredFiles {
		if now.After(expiry) {
			delete(fw.ignoredFiles, path)
		}
	}

	// Convert to absolute path for comparison
	absolutePath, _ := filepath.Abs(filePath)

	// Check if this file is currently being ignored
	if expiry, exists := fw.ignoredFiles[absolutePath]; exists && now.Before(expiry) {
		return true
	}

	return false
}

// broadcastFileChange sends file change notifications via WebSocket
func (fw *FileWatcher) broadcastFileChange(instanceName, filename string) {
	message := model.RspFileChange{
		Type:         "file_change",
		InstanceName: instanceName,
		Filename:     filename,
		Timestamp:    time.Now().Unix(),
	}
	wsManager := utils.GetWSManager()
	if wsManager != nil {
		wsManager.BroadcastJSON(message)
	}
}

// IsRunning returns whether the file watcher is currently active
func (fw *FileWatcher) IsRunning() bool {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	return fw.isRunning
}

// IgnoreFileWrite is a global function to mark a file as ignored before programmatic writes
func IgnoreFileWrite(instanceName string) {
	if fw := GetFileWatcher(); fw != nil && fw.IsRunning() {
		fw.IgnoreFile(instanceName)
	}
}
