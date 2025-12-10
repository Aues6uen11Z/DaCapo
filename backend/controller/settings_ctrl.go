package controller

import (
	"dacapo/backend/model"
	"dacapo/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetSettings retrieves application settings
func GetSettings(c *gin.Context) {
	settings, err := model.LoadSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load settings",
		})
		utils.Logger.Error("Failed to load settings:", err)
		return
	}

	response := model.RspSettings{
		Language:          settings.Language,
		RunOnStartup:      settings.RunOnStartup,
		SchedulerCron:     settings.SchedulerCron,
		AutoActionTrigger: settings.AutoActionTrigger,
		AutoActionCron:    settings.AutoActionCron,
		AutoActionType:    settings.AutoActionType,
		MaxBgConcurrent:   settings.MaxBgConcurrent,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateSettings updates application settings
func UpdateSettings(c *gin.Context) {
	var req model.ReqUpdateSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		utils.Logger.Error("Invalid request format:", err)
		return
	}

	if err := model.UpdateSettings(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save settings",
		})
		utils.Logger.Error("Failed to save settings:", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    model.StatusSuccess.Code,
		"message": model.StatusSuccess.Message,
		"detail":  "",
	})
}
