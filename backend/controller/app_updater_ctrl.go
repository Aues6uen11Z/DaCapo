package controller

import (
	"crypto/ed25519"
	"dacapo/backend/model"
	"dacapo/backend/utils"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/fynelabs/selfupdate"
	"github.com/gin-gonic/gin"
)

var (
	updateInProgress   bool
	updateMutex        sync.Mutex
	upgradeConfirmChan chan bool
	restartConfirmChan chan bool
	isManualUpdate     bool // Track if this is a manual update triggered by user
)

// CheckAppUpdate starts the update process directly with selfupdate, handles all interactions via WebSocket
func CheckAppUpdate(c *gin.Context) {
	// Check if this is a manual update request
	manual := c.Query("manual") == "true"

	updateMutex.Lock()
	if updateInProgress {
		updateMutex.Unlock()
		c.JSON(http.StatusOK, gin.H{
			"code":    model.StatusNetwork.Code,
			"message": model.StatusNetwork.Message,
			"detail":  "Update already in progress",
		})
		return
	}
	updateInProgress = true
	isManualUpdate = manual
	updateMutex.Unlock()

	// Initialize channels for user confirmations
	upgradeConfirmChan = make(chan bool, 1)
	restartConfirmChan = make(chan bool, 1)

	// Start the update process in a goroutine
	go func() {
		defer func() {
			updateMutex.Lock()
			updateInProgress = false
			isManualUpdate = false
			updateMutex.Unlock()
		}()

		utils.Logger.Info("Starting application update process...")
		currentVersion := utils.GetAppVersion()
		if currentVersion == "" {
			broadcastUpdateMessage("update_error", nil, "Application version not available")
			return
		}

		publicKey := ed25519.PublicKey{188, 205, 250, 191, 69, 34, 46, 0, 221, 241, 91, 230, 234, 166, 224, 161, 228, 14, 150, 61, 79, 28, 75, 97, 181, 215, 227, 244, 26, 110, 138, 109}
		updateSource := utils.NewUpdateSource(currentVersion)

		// Create selfupdate config with callbacks for all user interactions
		config := &selfupdate.Config{
			Source:    updateSource,
			PublicKey: publicKey,
			UpgradeConfirmCallback: func(message string) bool {
				// If this is a manual update (user clicked check updates), auto-confirm
				if isManualUpdate {
					utils.Logger.Info("Auto-confirming upgrade for manual update request")
					return true
				}

				// For auto-updates, ask frontend for confirmation via WebSocket
				broadcastUpdateMessage("update_confirm_upgrade", nil, message)
				select {
				case confirmed := <-upgradeConfirmChan:
					return confirmed
				case <-time.After(180 * time.Second):
					utils.Logger.Warn("Upgrade confirmation timeout, canceling update")
					broadcastUpdateMessage("update_error", nil, "Upgrade confirmation timeout")
					return false
				}
			},
			ProgressCallback: func(progress float64, err error) {
				if err != nil {
					broadcastUpdateMessage("update_error", nil, fmt.Sprintf("Progress error: %v", err))
					return
				}
				progressData := map[string]any{
					"progress":    progress,
					"description": fmt.Sprintf("Download progress: %.1f%%", progress*100),
				}
				broadcastUpdateMessage("update_progress", progressData, "")
			},
			RestartConfirmCallback: func() bool {
				// Ask frontend for restart confirmation via WebSocket
				broadcastUpdateMessage("update_confirm_restart", nil, "Update complete, restart now?")
				confirmed := <-restartConfirmChan
				if confirmed {
					broadcastUpdateMessage("update_restart_started", nil, "Restarting application...")
				}
				return confirmed
			},
		}

		// Create updater and proceed with the full update process
		updater, err := selfupdate.Manage(config)
		if err != nil {
			broadcastUpdateMessage("update_error", nil, fmt.Sprintf("Failed to create updater: %v", err))
			utils.Logger.Error("Failed to create updater:", err)
			return
		}

		// Proceed with the full update process
		err = updater.CheckNow()
		if err != nil {
			broadcastUpdateMessage("update_error", nil, fmt.Sprintf("Update process failed: %v", err))
			utils.Logger.Error("Failed to perform update:", err)
		} else {
			broadcastUpdateMessage("update_complete", nil, "Update process completed successfully")
			utils.Logger.Info("Self-update process completed")
		}
	}()
	c.JSON(http.StatusOK, gin.H{
		"code":    model.StatusSuccess.Code,
		"message": model.StatusSuccess.Message,
		"detail":  "Update process started",
	})
}

// HandleUpdateConfirmation handles user's confirmation for upgrade via WebSocket message
func HandleUpdateConfirmation(confirmed bool) {
	select {
	case upgradeConfirmChan <- confirmed:
		utils.Logger.Infof("Upgrade confirmation received: %v", confirmed)
	default:
		utils.Logger.Warn("No pending upgrade confirmation")
	}
}

// HandleRestartConfirmation handles user's confirmation for restart via WebSocket message
func HandleRestartConfirmation(confirmed bool) {
	select {
	case restartConfirmChan <- confirmed:
		utils.Logger.Infof("Restart confirmation received: %v", confirmed)
	default:
		utils.Logger.Warn("No pending restart confirmation")
	}
}

// broadcastUpdateMessage broadcasts update message via unified WebSocket
func broadcastUpdateMessage(msgType string, data any, message string) {
	wsManager := utils.GetWSManager()
	updateMsg := model.RspUpdateMessage{
		Type:    msgType,
		Data:    data,
		Message: message,
	}
	wsManager.BroadcastJSON(updateMsg)
}
