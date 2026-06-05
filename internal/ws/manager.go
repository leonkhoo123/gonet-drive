package ws

import (
	"encoding/json"
	"go-file-server/internal/logger"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WsManager handles websocket connections
type WsManager struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan interface{}
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
}

var (
	Manager = &WsManager{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan interface{}),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for now
		},
	}
)

// ClientMessage represents a message sent from the client
type ClientMessage struct {
	Type string `json:"type"`
	OpID string `json:"opId"`
}

// Start starts the websocket manager
func (m *WsManager) Start() {
	for {
		select {
		case client := <-m.register:
			m.mu.Lock()
			m.clients[client] = true
			m.mu.Unlock()
			logger.L.Info("websocket client connected")

		case client := <-m.unregister:
			m.mu.Lock()
			if _, ok := m.clients[client]; ok {
				delete(m.clients, client)
				client.Close()
				logger.L.Info("websocket client disconnected")
			}
			m.mu.Unlock()

		case message := <-m.broadcast:
			m.mu.Lock()
			for client := range m.clients {
				err := client.WriteJSON(message)
				if err != nil {
					logger.L.Error("websocket write error", "err", err)
					client.Close()
					delete(m.clients, client)
				}
			}
			m.mu.Unlock()
		}
	}
}

// Broadcast sends a message to all connected clients
func Broadcast(message interface{}) {
	Manager.broadcast <- message
}

// WsHandler handles the websocket handshake and manages real-time operation progress.
// @Summary      WebSocket Connection
// @Description  Establish a WebSocket connection for real-time file operation progress updates.
// @Tags         WebSocket
// @Security     BearerAuth
// @Security     CookieAuth
// @Success      101  {string}  string  "Switching Protocols"
// @Router       /api/user/ws [get]
func WsHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.L.Error("failed to upgrade to websocket", "err", err)
		return
	}

	Manager.register <- conn

	// Keep connection alive/check for disconnection
	go func() {
		defer func() {
			Manager.unregister <- conn
		}()

		// Set an initial read deadline. We expect a ping every 30s from the client.
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// If the client sends an actual WebSocket PING frame
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		for {
			_, p, err := conn.ReadMessage()
			if err != nil {
				return
			}

			// Extend the deadline whenever we receive ANY message
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			// Handle client messages
			var msg ClientMessage
			if err := json.Unmarshal(p, &msg); err != nil {
				logger.L.Error("failed to unmarshal websocket message", "err", err)
				continue
			}

			if msg.Type == "ping" {
				// Heartbeat message to keep connection alive
				continue
			}

			if msg.Type == "check_progress" && msg.OpID != "" {
				if opMeta, found := GetOpMeta(msg.OpID); found {
					Broadcast(opMeta)
				}
			}
		}
	}()
}

// OperationMessage is the standardized structure for all file operation updates
type OperationMessage struct {
	OpId         string   `json:"opId"`
	OpType       string   `json:"opType"`            // "copy", "move", "delete"
	OpName       string   `json:"opName,omitempty"`  // Brief description, e.g. "Copying file name to destination"
	OpStatus     string   `json:"opStatus"`          // "starting", "in-progress", "error", "completed", "aborted"
	OpPercentage *float64 `json:"opPercentage"`      // 0-100, null when error/completed/aborted
	OpSpeed      *string  `json:"opSpeed,omitempty"` // only for copy, null otherwise
	OpFileCount  *string  `json:"opFileCount"`       // "3/367", null when error/completed/aborted
	Error        *string  `json:"error,omitempty"`   // error message if status is "error" or "aborted"
	DestDir      *string  `json:"destDir,omitempty"` // destination directory
	RequestId    *string  `json:"requestId,omitempty"`
}
