package ws

import (
	"encoding/json"
	"fmt"
	"go-file-server/internal/state"
	"go-file-server/internal/util"
	"log"
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
			log.Println("New WebSocket client connected")

		case client := <-m.unregister:
			m.mu.Lock()
			if _, ok := m.clients[client]; ok {
				delete(m.clients, client)
				client.Close()
				log.Println("WebSocket client disconnected")
			}
			m.mu.Unlock()

		case message := <-m.broadcast:
			m.mu.Lock()
			for client := range m.clients {
				err := client.WriteJSON(message)
				if err != nil {
					log.Printf("Websocket error: %v", err)
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

// WsHandler handles the websocket handshake
func WsHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to websocket: %v", err)
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
				log.Printf("Error unmarshalling client message: %v", err)
				continue
			}

			if msg.Type == "ping" {
				// Heartbeat message to keep connection alive
				continue
			}

			if msg.Type == "check_progress" && msg.OpID != "" {
				if pt, found := state.GetProgress(msg.OpID); found {
					// This handler is kept for backwards compatibility
					// but ideally clients should rely on broadcasted messages
					// Note: We don't know the opType here, so we can't populate it
					// This is a limitation of the check_progress pattern
					// Clients should track opType themselves or we need to store it with the tracker

					percentage := 0.0
					if pt.TotalFiles > 0 {
						if pt.TotalBytes > 0 {
							percentage = float64(pt.CopiedBytes) / float64(pt.TotalBytes) * 100
						} else {
							percentage = float64(pt.CopiedFiles) / float64(pt.TotalFiles) * 100
						}
					}

					var speedStr *string
					elapsed := time.Since(pt.StartTime).Seconds()
					if elapsed > 0 && pt.CopiedBytes > 0 {
						speed := float64(pt.CopiedBytes) / elapsed
						s := util.FormatBytes(int64(speed)) + "/s"
						speedStr = &s
					}

					fileCountStr := fmt.Sprintf("%d/%d", pt.CopiedFiles, pt.TotalFiles)

					response := OperationMessage{
						OpId:         msg.OpID,
						OpType:       "", // Unknown in this context
						OpStatus:     "in-progress",
						OpPercentage: &percentage,
						OpSpeed:      speedStr,
						OpFileCount:  &fileCountStr,
					}

					Broadcast(response)
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
}
