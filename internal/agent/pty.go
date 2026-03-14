package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

// PTYSession represents an active pseudo-terminal session.
type PTYSession struct {
	ID   string
	File *os.File
	Cmd  *exec.Cmd
}

// PTYManager manages active PTY sessions.
type PTYManager struct {
	sessions map[string]*PTYSession
	mu       sync.Mutex
	nextID   int
}

// NewPTYManager creates a new PTY manager.
func NewPTYManager() *PTYManager {
	return &PTYManager{
		sessions: make(map[string]*PTYSession),
	}
}

// Spawn creates a new PTY session running a shell as the given user.
func (m *PTYManager) Spawn(user string, cols, rows uint16) (*PTYSession, error) {
	var cmd *exec.Cmd
	if user == "" || user == "root" {
		cmd = exec.Command("/bin/bash", "-l")
	} else {
		cmd = exec.Command("su", "-l", user)
	}

	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: rows, Cols: cols})
	if err != nil {
		return nil, fmt.Errorf("starting pty: %w", err)
	}

	m.mu.Lock()
	m.nextID++
	id := fmt.Sprintf("pty-%d", m.nextID)
	session := &PTYSession{
		ID:   id,
		File: ptmx,
		Cmd:  cmd,
	}
	m.sessions[id] = session
	m.mu.Unlock()

	return session, nil
}

// Get returns a session by ID.
func (m *PTYManager) Get(id string) (*PTYSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return s, nil
}

// Close terminates a PTY session.
func (m *PTYManager) Close(id string) {
	m.mu.Lock()
	s, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
	}
	m.mu.Unlock()

	if ok && s != nil {
		s.File.Close()
		if s.Cmd.Process != nil {
			s.Cmd.Process.Kill()
		}
		s.Cmd.Wait()
	}
}

// Resize changes the terminal size of a session.
func (m *PTYManager) Resize(id string, cols, rows uint16) error {
	s, err := m.Get(id)
	if err != nil {
		return err
	}
	return pty.Setsize(s.File, &pty.Winsize{Rows: rows, Cols: cols})
}

// CloseAll terminates all active sessions.
func (m *PTYManager) CloseAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	for _, id := range ids {
		m.Close(id)
	}
}

// ptyResizeParams are the parameters for the pty_resize command.
type ptyResizeParams struct {
	SessionID string `json:"session_id"`
	Cols      uint16 `json:"cols"`
	Rows      uint16 `json:"rows"`
}

// cmdPtyResize handles the pty_resize command via normal JSON-RPC.
func cmdPtyResize(ptyMgr *PTYManager) CommandFunc {
	return func(params json.RawMessage) (interface{}, error) {
		var p ptyResizeParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if err := ptyMgr.Resize(p.SessionID, p.Cols, p.Rows); err != nil {
			return nil, err
		}
		return map[string]string{"status": "ok"}, nil
	}
}
