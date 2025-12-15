package controller

import (
	"dacapo/backend/model"
	"dacapo/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// CreateWS establishes a unified WebSocket connection for all message types
func CreateWS(c *gin.Context) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		utils.Logger.Error("Failed to upgrade websocket: ", err)
		return
	}
	// Create message handler function for app update responses
	messageHandler := func(msgType string, data map[string]any) {
		if msgType == "update_confirm_response" {
			if confirmed, ok := data["confirmed"].(bool); ok {
				HandleUpdateConfirmation(confirmed)
			}
		} else if msgType == "restart_confirm_response" {
			if confirmed, ok := data["confirmed"].(bool); ok {
				HandleRestartConfirmation(confirmed)
			}
		}
	}

	Services.WebSocketService().HandleConnection(conn, messageHandler)
}

// UpdateTaskQueue updates the task queue
func UpdateTaskQueue(c *gin.Context) {
	var req model.ReqUpdateQueue
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		utils.Logger.Error("Invalid request format\n", err)
		return
	}

	Services.SchedulerService().UpdateTaskQueue(req.Queues)

	c.JSON(http.StatusOK, gin.H{
		"code":    model.StatusSuccess.Code,
		"message": model.StatusSuccess.Message,
		"detail":  "",
	})
}

// UpdateSchedulerState updates the scheduler state
func UpdateSchedulerState(c *gin.Context) {
	var req model.ReqSchedulerState
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		utils.Logger.Error("Invalid request format\n", err)
		return
	}

	Services.SchedulerService().UpdateSchedulerState(req.Type, req.InstanceName)

	c.JSON(http.StatusOK, gin.H{
		"code":    model.StatusSuccess.Code,
		"message": model.StatusSuccess.Message,
		"detail":  "",
	})
}

func GetTaskQueue(c *gin.Context) {
	instanceName := c.Param("instance_name")
	Services.SchedulerService().GetTaskQueue(instanceName)

	c.JSON(http.StatusOK, gin.H{
		"code":    model.StatusSuccess.Code,
		"message": model.StatusSuccess.Message,
		"detail":  "",
	})
}

func SetSchedulerCron(c *gin.Context) {
	var req model.ReqSchedulerCron
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		utils.Logger.Error("Invalid request format\n", err)
		return
	}

	Services.SchedulerService().SetSchedulerCron(req.CronExpr)

	c.JSON(http.StatusOK, gin.H{
		"code":    model.StatusSuccess.Code,
		"message": model.StatusSuccess.Message,
		"detail":  "",
	})
}
