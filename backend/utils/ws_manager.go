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

var (
	wsManager *WSManager
	initMutex sync.Mutex
)

// GetWSManager returns the singleton instance of WebSocket manager
func GetWSManager() *WSManager {
	initMutex.Lock()
	defer initMutex.Unlock()
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
	Logger.Infof("New WebSocket client %s registered, total: %d", conn.RemoteAddr(), len(m.clients))
}

// RemoveClient removes a WebSocket connection
func (m *WSManager) RemoveClient(conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, conn)
	Logger.Infof("WebSocket client %s removed, total: %d", conn.RemoteAddr(), len(m.clients))
}

// BroadcastJSON broadcasts JSON message to all clients
func (m *WSManager) BroadcastJSON(message any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var failedClients []*websocket.Conn
	for conn := range m.clients {
		if err := conn.WriteJSON(message); err != nil {
			Logger.Errorf("Failed to send WebSocket message to %s: %v", conn.RemoteAddr(), err)
			failedClients = append(failedClients, conn)
		}
	}

	// Clean up failed connections
	for _, conn := range failedClients {
		conn.Close()
		delete(m.clients, conn)
	}

	if len(failedClients) > 0 {
		Logger.Infof("Cleaned up %d failed WebSocket connections, remaining: %d", len(failedClients), len(m.clients))
	}
}

// SendJSON sends JSON message to a specific client
func (m *WSManager) SendJSON(conn *websocket.Conn, message any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := conn.WriteJSON(message); err != nil {
		Logger.Errorf("Failed to send WebSocket message to %s: %v", conn.RemoteAddr(), err)
		return err
	}
	return nil
}
