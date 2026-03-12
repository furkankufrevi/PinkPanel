// Package websocket provides a WebSocket hub for real-time dashboard metrics.
package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
)

// Hub manages WebSocket connections and broadcasts system metrics.
type Hub struct {
	clients     map[*websocket.Conn]bool
	mu          sync.RWMutex
	agentClient *agent.Client
	stop        chan struct{}
}

// NewHub creates a new WebSocket hub.
func NewHub(agentClient *agent.Client) *Hub {
	return &Hub{
		clients:     make(map[*websocket.Conn]bool),
		agentClient: agentClient,
		stop:        make(chan struct{}),
	}
}

// HandleConnection handles a new WebSocket connection.
func (h *Hub) HandleConnection(c *websocket.Conn) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, c)
		h.mu.Unlock()
		c.Close()
	}()

	// Keep connection alive by reading (client may send pings)
	for {
		if _, _, err := c.ReadMessage(); err != nil {
			break
		}
	}
}

// Start begins broadcasting metrics to all connected clients every 5 seconds.
func (h *Hub) Start() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				h.broadcast()
			case <-h.stop:
				return
			}
		}
	}()
	log.Info().Msg("WebSocket metrics hub started (broadcasts every 5s)")
}

// Stop halts the hub.
func (h *Hub) Stop() {
	close(h.stop)
	h.mu.Lock()
	for c := range h.clients {
		c.Close()
	}
	h.mu.Unlock()
}

func (h *Hub) broadcast() {
	h.mu.RLock()
	count := len(h.clients)
	h.mu.RUnlock()

	if count == 0 {
		return // No clients, skip agent call
	}

	resp, err := h.agentClient.Call("system_info", nil)
	if err != nil {
		return
	}

	data, err := json.Marshal(resp.Result)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			// Will be cleaned up on next read failure
			log.Debug().Err(err).Msg("ws write failed")
		}
	}
}
