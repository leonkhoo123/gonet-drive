package ws

import (
	"encoding/json"
	"go-file-server/internal/logger"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	authgin "github.com/leonkhoo123/gonet-auth/adapters/gin"
	"github.com/gorilla/websocket"
)

// WsManager handles websocket connections
type WsManager struct {
	clients    map[*websocket.Conn]string // conn -> username
	broadcast  chan interface{}
	register   chan *wsClient
	unregister chan *websocket.Conn
	mu         sync.RWMutex
}

// wsClient pairs a connection with its authenticated username
type wsClient struct {
	conn     *websocket.Conn
	username string
}

var (
	Manager = &WsManager{
		clients:    make(map[*websocket.Conn]string),
		broadcast:  make(chan interface{}, 64),
		register:   make(chan *wsClient, 8),
		unregister: make(chan *websocket.Conn, 8),
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
			m.clients[client.conn] = client.username
			m.mu.Unlock()
			logger.L.Info("websocket client connected", "username", client.username)

		case conn := <-m.unregister:
			m.mu.Lock()
			if _, ok := m.clients[conn]; ok {
				delete(m.clients, conn)
				conn.Close()
				logger.L.Info("websocket client disconnected")
			}
			m.mu.Unlock()

		case message := <-m.broadcast:
			m.mu.RLock()
			for conn, username := range m.clients {
				_ = username
				err := conn.WriteJSON(message)
				if err != nil {
					logger.L.Error("websocket write error", "err", err)
					conn.Close()
					delete(m.clients, conn)
				}
			}
			m.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients
func Broadcast(message interface{}) {
	select {
	case Manager.broadcast <- message:
	default:
		logger.L.Warn("websocket broadcast channel full, dropping message")
	}
}

// BroadcastToUser sends a message only to connections belonging to a specific username.
// Falls back to global broadcast if username is empty.
func BroadcastToUser(username string, message interface{}) {
	if username == "" {
		Broadcast(message)
		return
	}
	Manager.mu.RLock()
	defer Manager.mu.RUnlock()
	for conn, user := range Manager.clients {
		if user == username {
			err := conn.WriteJSON(message)
			if err != nil {
				logger.L.Error("websocket write error", "err", err)
				conn.Close()
				delete(Manager.clients, conn)
			}
		}
	}
}

// resolveUsername determines the identity used for per-user broadcasts.
// Authenticated user connections carry authgin.KeyUsername, while share
// connections carry share_id (matching the "share:<id>" owner used by the
// share file service). Falls back to the legacy "username" context key.
func resolveUsername(c *gin.Context) string {
	if username := c.GetString(authgin.KeyUsername); username != "" {
		return username
	}
	if shareID := c.GetString("share_id"); shareID != "" {
		return "share:" + shareID
	}
	return c.GetString("username")
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

	username := resolveUsername(c)

	Manager.register <- &wsClient{conn: conn, username: username}

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
					BroadcastToUser(username, opMeta)
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
