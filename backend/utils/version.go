package utils

import (
	"sync"
)

var (
	appVersion string
	versionMu  sync.RWMutex
)

// SetAppVersion sets the application version
func SetAppVersion(version string) {
	versionMu.Lock()
	defer versionMu.Unlock()
	appVersion = version
}

// GetAppVersion gets the application version
func GetAppVersion() string {
	versionMu.RLock()
	defer versionMu.RUnlock()
	return appVersion
}
