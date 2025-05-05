package main

import (
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

	r := router.SetupRouter()
	r.Run(":48596")
}
