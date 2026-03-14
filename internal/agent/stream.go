package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// StreamConn is a dedicated connection to the agent for raw PTY streaming.
// Unlike Client (which is mutex-locked and shared), each StreamConn is used
// by a single terminal session.
type StreamConn struct {
	conn      net.Conn
	SessionID string
}

// DialStream opens a new Unix socket connection to the agent and performs
// the PTY spawn handshake. After this call, the connection is in raw stream
// mode — reads return PTY output, writes go to PTY input.
func DialStream(socketPath, user string, cols, rows uint16) (*StreamConn, error) {
	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("connecting to agent: %w", err)
	}

	// Send pty_spawn JSON-RPC request
	req := Request{
		Method: "pty_spawn",
		ID:     1,
	}
	params, _ := json.Marshal(map[string]interface{}{
		"user": user,
		"cols": cols,
		"rows": rows,
	})
	req.Params = params

	data, _ := json.Marshal(req)
	data = append(data, '\n')

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write(data); err != nil {
		conn.Close()
		return nil, fmt.Errorf("writing pty_spawn request: %w", err)
	}

	// Read the JSON-RPC response (the last JSON message before raw mode)
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		conn.Close()
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading pty_spawn response: %w", err)
		}
		return nil, fmt.Errorf("connection closed during handshake")
	}

	var resp Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		conn.Close()
		return nil, fmt.Errorf("parsing pty_spawn response: %w", err)
	}
	if resp.Error != "" {
		conn.Close()
		return nil, fmt.Errorf("pty_spawn failed: %s", resp.Error)
	}

	// Extract session ID
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		conn.Close()
		return nil, fmt.Errorf("unexpected response format")
	}
	sessionID, _ := result["session_id"].(string)
	if sessionID == "" {
		conn.Close()
		return nil, fmt.Errorf("no session_id in response")
	}

	// Clear deadlines — the connection is now a raw stream
	conn.SetReadDeadline(time.Time{})
	conn.SetWriteDeadline(time.Time{})

	return &StreamConn{
		conn:      conn,
		SessionID: sessionID,
	}, nil
}

// Read reads raw PTY output from the stream.
func (s *StreamConn) Read(p []byte) (int, error) {
	return s.conn.Read(p)
}

// Write writes raw input to the PTY via the stream.
func (s *StreamConn) Write(p []byte) (int, error) {
	return s.conn.Write(p)
}

// Close closes the stream connection.
func (s *StreamConn) Close() error {
	return s.conn.Close()
}
