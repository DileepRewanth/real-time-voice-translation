package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/voice-translation/backend/internal/domain"
	"github.com/voice-translation/backend/internal/pipeline"
)

// --- WebSocket Hub ---

// Hub manages all active WebSocket client connections.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	logger     *slog.Logger
}

// NewHub creates a new WebSocket hub.
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

// Run starts the hub's main event loop.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Info("websocket client connected",
				"total_clients", len(h.clients),
			)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			h.logger.Info("websocket client disconnected",
				"total_clients", len(h.clients),
			)
		}
	}
}

// --- WebSocket Client ---

// Client represents a single WebSocket connection.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	pipe   *pipeline.Pipeline
	logger *slog.Logger
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 8192
)

// readPump listens for incoming WebSocket messages.
// Runs in its own goroutine per client.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.logger.Error("websocket read error", "error", err)
			}
			break
		}

		c.handleMessage(message)
	}
}

// writePump sends messages to the WebSocket connection.
// Runs in its own goroutine per client.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.logger.Error("websocket write error", "error", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages.
func (c *Client) handleMessage(data []byte) {
	var msg domain.WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("Invalid message format")
		return
	}

	switch msg.Type {
	case "translate":
		c.handleTranslate(msg.Payload)
	case "ping":
		c.sendJSON("pong", map[string]interface{}{
			"timestamp": time.Now().UnixMilli(),
		})
	default:
		c.sendError("Unknown message type: " + msg.Type)
	}
}

// handleTranslate processes a translation request via the async pipeline.
func (c *Client) handleTranslate(payload interface{}) {
	// Parse payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		c.sendError("Invalid payload")
		return
	}

	var req domain.WSTranslatePayload
	if err := json.Unmarshal(payloadBytes, &req); err != nil {
		c.sendError("Invalid translate payload")
		return
	}

	if req.Text == "" {
		c.sendError("Text field is required")
		return
	}

	// Build domain request
	engine := domain.EngineMyMemory
	if req.Engine == "gemini" {
		engine = domain.EngineGemini
	}

	domainReq := domain.TranslationRequest{
		Text:           req.Text,
		Engine:         engine,
		Context:        req.Context,
		TonePreference: req.TonePreference,
	}

	// Run async pipeline with stage updates
	statusCh := make(chan domain.WSStatusPayload, 10)
	resultCh := make(chan *domain.TranslationResult, 1)
	errCh := make(chan error, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	c.pipe.ProcessAsync(ctx, domainReq, statusCh, resultCh, errCh)

	// Stream stage updates back to the client
	go func() {
		defer cancel()
		for {
			select {
			case status, ok := <-statusCh:
				if !ok {
					return
				}
				c.sendJSON("status", status)

			case result := <-resultCh:
				c.sendJSON("translation", result)
				return

			case err := <-errCh:
				c.sendError(err.Error())
				return

			case <-ctx.Done():
				c.sendError("Translation timed out")
				return
			}
		}
	}()
}

// sendJSON sends a typed JSON message to the client.
func (c *Client) sendJSON(msgType string, payload interface{}) {
	msg := domain.WSMessage{
		Type:    msgType,
		Payload: payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		c.logger.Error("failed to marshal ws message", "error", err)
		return
	}

	select {
	case c.send <- data:
	default:
		c.logger.Warn("client send buffer full, dropping message")
	}
}

// sendError sends an error message to the client.
func (c *Client) sendError(message string) {
	c.sendJSON("error", map[string]string{
		"message": message,
	})
}

// --- WebSocket Handler ---

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Configured per deployment — for dev, allow all
	},
}

// Global hub instance (initialized in main.go via SetHub)
var globalHub *Hub

// SetHub sets the global WebSocket hub reference.
func SetHub(h *Hub) {
	globalHub = h
}

// wsHandler upgrades HTTP connections to WebSocket.
func wsHandler(pipe *pipeline.Pipeline, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("websocket upgrade failed", "error", err)
			return
		}

		client := &Client{
			hub:    globalHub,
			conn:   conn,
			send:   make(chan []byte, 256),
			pipe:   pipe,
			logger: logger,
		}

		globalHub.register <- client

		// Start read/write pumps in separate goroutines (decoupled loop pattern)
		go client.writePump()
		go client.readPump()
	}
}
