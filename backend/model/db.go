package model

import (
	"dacapo/backend/utils"

	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func InitDB() {
	var err error
	db, err = gorm.Open(gormlite.Open("dacapo.db"), &gorm.Config{})
	if err != nil {
		utils.Logger.Fatal("Failed to connect database: ", err)
	}

	err = db.AutoMigrate(
		&TemplateInfo{},
		&InstanceInfo{},
		&TaskInfo{},
	)
	if err != nil {
		utils.Logger.Fatal("Failed to migrate database: ", err)
	}

	utils.Logger.Info("Database initialized")
}
