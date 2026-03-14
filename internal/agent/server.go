package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
)

// Request is the JSON-RPC style request sent to the agent.
type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
	ID     int             `json:"id,omitempty"`
}

// Response is the JSON-RPC style response from the agent.
type Response struct {
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
	ID     int         `json:"id,omitempty"`
}

// Server is the Unix socket JSON-RPC agent server.
type Server struct {
	socketPath string
	listener   net.Listener
	commands   *CommandRegistry
	ptyMgr     *PTYManager
	mu         sync.Mutex
}

// NewServer creates a new agent server.
func NewServer(socketPath string) *Server {
	ptyMgr := NewPTYManager()
	return &Server{
		socketPath: socketPath,
		commands:   NewCommandRegistry(),
		ptyMgr:     ptyMgr,
	}
}

// Start begins listening on the Unix socket.
func (s *Server) Start() error {
	// Register PTY commands that need the manager
	s.commands.commands["pty_resize"] = cmdPtyResize(s.ptyMgr)

	// Create socket directory
	dir := s.socketPath[:len(s.socketPath)-len("/agent.sock")]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating socket directory: %w", err)
	}

	// Remove stale socket
	os.Remove(s.socketPath)

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listening on socket: %w", err)
	}

	// Allow non-root processes (pinkpanel user) to connect to the socket
	if err := os.Chmod(s.socketPath, 0666); err != nil {
		return fmt.Errorf("setting socket permissions: %w", err)
	}

	s.mu.Lock()
	s.listener = listener
	s.mu.Unlock()

	log.Printf("Agent listening on %s", s.socketPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			// Check if we're shutting down
			s.mu.Lock()
			l := s.listener
			s.mu.Unlock()
			if l == nil {
				return nil
			}
			log.Printf("Accept error: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

// Stop closes the listener and cleans up the socket.
func (s *Server) Stop() {
	s.ptyMgr.CloseAll()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		s.listener.Close()
		s.listener = nil
	}
	os.Remove(s.socketPath)
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var req Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			writeResponse(conn, Response{Error: "invalid JSON", ID: req.ID})
			continue
		}

		// Handle pty_spawn specially — after response, switch to raw stream mode
		if req.Method == "pty_spawn" {
			s.handlePtySpawn(conn, req)
			return // connection is now in stream mode, exit the JSON-RPC loop
		}

		resp := s.dispatch(req)
		writeResponse(conn, resp)
	}
}

func (s *Server) handlePtySpawn(conn net.Conn, req Request) {
	var params struct {
		User string `json:"user"`
		Cols uint16 `json:"cols"`
		Rows uint16 `json:"rows"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeResponse(conn, Response{Error: "invalid params: " + err.Error(), ID: req.ID})
		return
	}
	if params.Cols == 0 {
		params.Cols = 80
	}
	if params.Rows == 0 {
		params.Rows = 24
	}

	session, err := s.ptyMgr.Spawn(params.User, params.Cols, params.Rows)
	if err != nil {
		writeResponse(conn, Response{Error: err.Error(), ID: req.ID})
		return
	}

	// Send success response with session ID and stream flag
	writeResponse(conn, Response{
		Result: map[string]interface{}{
			"session_id": session.ID,
			"stream":     true,
		},
		ID: req.ID,
	})

	// Enter raw stream mode: bridge PTY <-> socket bidirectionally
	done := make(chan struct{}, 2)

	// PTY → socket
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 4096)
		for {
			n, err := session.File.Read(buf)
			if n > 0 {
				if _, werr := conn.Write(buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// Socket → PTY
	go func() {
		defer func() { done <- struct{}{} }()
		io.Copy(session.File, conn)
	}()

	// Wait for either direction to finish
	<-done

	// Clean up
	s.ptyMgr.Close(session.ID)
}

func (s *Server) dispatch(req Request) Response {
	resp := Response{ID: req.ID}

	result, err := s.commands.Execute(req.Method, req.Params)
	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Result = result
	}

	return resp
}

func writeResponse(conn net.Conn, resp Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}
	data = append(data, '\n')
	conn.Write(data)
}
