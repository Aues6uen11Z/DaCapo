package main

import (
	"dacapo/backend/controller"
	"dacapo/backend/model"
	"dacapo/backend/router"
	"dacapo/backend/utils"
)

func main() {
	utils.InitLogger()
	defer utils.Logger.Sync()

	model.InitDB()

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

	r := router.SetupRouter()
	r.Run(":48596")
}
