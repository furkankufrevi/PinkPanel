package handlers

import (
	"encoding/json"

	"github.com/gofiber/websocket/v2"
	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/auth"
)

// TerminalHandler handles WebSocket terminal connections.
type TerminalHandler struct {
	AgentSocketPath string
	JWTManager      *auth.JWTManager
	AgentClient     *agent.Client
}

// HandleTerminal bridges a WebSocket connection to an agent PTY session.
func (h *TerminalHandler) HandleTerminal(c *websocket.Conn) {
	defer c.Close()

	// Auth via query param (WebSocket can't use Authorization header)
	token := c.Query("token")
	if token == "" {
		c.WriteJSON(map[string]string{"type": "error", "message": "missing token"})
		return
	}

	claims, err := h.JWTManager.ValidateAccessToken(token)
	if err != nil {
		c.WriteJSON(map[string]string{"type": "error", "message": "invalid token"})
		return
	}

	// Only admins can use terminal
	if claims.Role != "super_admin" && claims.Role != "admin" {
		c.WriteJSON(map[string]string{"type": "error", "message": "insufficient permissions"})
		return
	}

	// Read initial config from client
	_, msg, err := c.ReadMessage()
	if err != nil {
		return
	}

	var initConfig struct {
		Cols uint16 `json:"cols"`
		Rows uint16 `json:"rows"`
	}
	if err := json.Unmarshal(msg, &initConfig); err != nil {
		c.WriteJSON(map[string]string{"type": "error", "message": "invalid init config"})
		return
	}
	if initConfig.Cols == 0 {
		initConfig.Cols = 80
	}
	if initConfig.Rows == 0 {
		initConfig.Rows = 24
	}

	// Open a dedicated stream connection to the agent
	stream, err := agent.DialStream(h.AgentSocketPath, "root", initConfig.Cols, initConfig.Rows)
	if err != nil {
		log.Error().Err(err).Msg("terminal: failed to open PTY stream")
		c.WriteJSON(map[string]string{"type": "error", "message": "failed to start terminal: " + err.Error()})
		return
	}
	defer stream.Close()

	done := make(chan struct{}, 2)

	// Agent PTY → WebSocket (binary output)
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 4096)
		for {
			n, err := stream.Read(buf)
			if n > 0 {
				if werr := c.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// WebSocket → Agent PTY (input + resize)
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			msgType, data, err := c.ReadMessage()
			if err != nil {
				return
			}

			// Check if this is a JSON control message (resize)
			if msgType == websocket.TextMessage {
				var ctrl struct {
					Type string `json:"type"`
					Cols uint16 `json:"cols"`
					Rows uint16 `json:"rows"`
				}
				if json.Unmarshal(data, &ctrl) == nil && ctrl.Type == "resize" {
					// Use a normal JSON-RPC call to resize (separate from stream)
					h.AgentClient.Call("pty_resize", map[string]interface{}{
						"session_id": stream.SessionID,
						"cols":       ctrl.Cols,
						"rows":       ctrl.Rows,
					})
					continue
				}
			}

			// Raw input → PTY
			if _, err := stream.Write(data); err != nil {
				return
			}
		}
	}()

	<-done
}
