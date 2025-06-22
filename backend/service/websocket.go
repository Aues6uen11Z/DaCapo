package service

import (
	"dacapo/backend/model"
	"dacapo/backend/utils"

	"github.com/gorilla/websocket"
)

type WebSocketService struct{}

// HandleConnection handles the unified WebSocket connection
func (s *WebSocketService) HandleConnection(conn *websocket.Conn, messageHandler func(string, map[string]any)) {
	defer conn.Close()

	wsManager := utils.GetWSManager()
	wsManager.RegisterClient(conn)
	defer wsManager.RemoveClient(conn)

	// Send initial state
	s.SendQueue(conn)
	s.SendState(conn)

	// Listen for messages from client
	for {
		var msg map[string]any
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		// Handle different message types
		if msgType, ok := msg["type"].(string); ok {
			if data, ok := msg["data"].(map[string]any); ok {
				// Handle app update related messages
				if msgType == "update_confirm_response" || msgType == "restart_confirm_response" {
					messageHandler(msgType, data)
				}
			}
		}
	}
}

// SendQueue sends the task queue to a specific connection
func (s *WebSocketService) SendQueue(conn *websocket.Conn) {
	scheduler := model.GetScheduler()
	queues := scheduler.GetTaskQueues()
	for istName, queue := range queues {
		update := model.RspTaskQueue{
			Type:         "queue",
			InstanceName: istName,
			Queue:        queue,
		}
		if err := utils.GetWSManager().SendJSON(conn, update); err != nil {
			return
		}
	}
}

// SendState sends the scheduler state to a specific connection
func (s *WebSocketService) SendState(conn *websocket.Conn) {
	scheduler := model.GetScheduler()
	for istName, tm := range scheduler.TaskManagers {
		instanceState := model.RspSchedulerState{
			Type:         "state",
			InstanceName: istName,
			State:        tm.Status,
		}
		if err := utils.GetWSManager().SendJSON(conn, instanceState); err != nil {
			return
		}
	}
	schedulerState := model.RspSchedulerState{
		Type:         "state",
		InstanceName: "",
		State:        map[bool]string{true: "running", false: "pending"}[scheduler.IsRunning],
	}
	utils.GetWSManager().SendJSON(conn, schedulerState)
}

// BroadcastQueue broadcasts queue updates
func (s *WebSocketService) BroadcastQueue(istName string) {
	scheduler := model.GetScheduler()
	tm := scheduler.GetTaskManager(istName)
	if tm == nil {
		return
	}
	update := model.RspTaskQueue{
		Type:         "queue",
		InstanceName: istName,
		Queue:        tm.Queue,
	}
	utils.GetWSManager().BroadcastJSON(update)
}

// BroadcastState broadcasts scheduler state
func (s *WebSocketService) BroadcastState(istName, state string) {
	message := model.RspSchedulerState{
		Type:         "state",
		InstanceName: istName,
		State:        state,
	}
	utils.GetWSManager().BroadcastJSON(message)
}

// BroadcastLog broadcasts log messages
func (s *WebSocketService) BroadcastLog(istName, content string) {
	message := model.RspLogMessage{
		Type:         "log",
		InstanceName: istName,
		Content:      content,
	}
	utils.GetWSManager().BroadcastJSON(message)
}
