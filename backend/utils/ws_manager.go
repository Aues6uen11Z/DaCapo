package utils

import (
	"sync"

	"github.com/gorilla/websocket"
)

// WSManager manages all WebSocket connections
type WSManager struct {
	clients map[*websocket.Conn]bool
	mu      sync.Mutex // Mutex for thread safety
}

var wsManager *WSManager

// GetWSManager returns the singleton instance of WebSocket manager
func GetWSManager() *WSManager {
	if wsManager == nil {
		wsManager = &WSManager{
			clients: make(map[*websocket.Conn]bool),
		}
	}
	return wsManager
}

// RegisterClient registers a new WebSocket connection
func (m *WSManager) RegisterClient(conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[conn] = true
	Logger.Infof("New client %s registered, total: %d", conn.RemoteAddr(), len(m.clients))
}

// RemoveClient removes a WebSocket connection
func (m *WSManager) RemoveClient(conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, conn)
	Logger.Infof("Client %s removed, total: %d", conn.RemoteAddr(), len(m.clients))
}

// BroadcastJSON broadcasts JSON message to all clients
func (m *WSManager) BroadcastJSON(message any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for conn := range m.clients {
		if err := conn.WriteJSON(message); err != nil {
			Logger.Errorf("Failed to send message to %s: %v", conn.RemoteAddr(), err)
			conn.Close()
			delete(m.clients, conn)
		}
	}
}

// SendJSON sends JSON message to a specific client
func (m *WSManager) SendJSON(conn *websocket.Conn, message any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := conn.WriteJSON(message); err != nil {
		Logger.Errorf("Failed to send message to %s: %v", conn.RemoteAddr(), err)
		return err
	}
	return nil
}
