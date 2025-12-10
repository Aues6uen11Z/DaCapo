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

	// Migrate order for existing instances
	if err := migrateOrder(); err != nil {
		utils.Logger.Warnf("Failed to migrate order: %v", err)
	}

	utils.Logger.Info("Database initialized")
}

// migrateOrder initializes order for existing instances
func migrateOrder() error {
	var count int64
	db.Model(&InstanceInfo{}).Where("`order` = -1").Count(&count)

	if count == 0 {
		return nil
	}

	utils.Logger.Infof("Migrating order for %d instances...", count)

	var instances []InstanceInfo
	if err := db.Where("`order` = -1").Order("id ASC").Find(&instances).Error; err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for i, instance := range instances {
			if err := tx.Model(&instance).Update("`order`", i).Error; err != nil {
				return err
			}
		}
		utils.Logger.Infof("Migrated order for %d instances", len(instances))
		return nil
	})
}

// CloseDB closes the database connection
func CloseDB() {
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			utils.Logger.Errorf("Failed to get underlying sql.DB: %v", err)
			return
		}

		err = sqlDB.Close()
		if err != nil {
			utils.Logger.Errorf("Failed to close database: %v", err)
		} else {
			utils.Logger.Info("Database connection closed")
		}
	}
}
