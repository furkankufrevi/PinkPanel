package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Client communicates with the agent over a Unix socket.
type Client struct {
	socketPath string
	conn       net.Conn
	scanner    *bufio.Scanner
	mu         sync.Mutex
	nextID     atomic.Int64
}

// NewClient creates a new agent client.
func NewClient(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
	}
}

// Connect establishes a connection to the agent socket.
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := net.DialTimeout("unix", c.socketPath, 5*time.Second)
	if err != nil {
		return fmt.Errorf("connecting to agent: %w", err)
	}

	c.conn = conn
	c.scanner = bufio.NewScanner(conn)
	return nil
}

// Close closes the connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Call sends a request and waits for the response.
func (c *Client) Call(method string, params interface{}) (*Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, fmt.Errorf("not connected to agent")
	}

	id := int(c.nextID.Add(1))

	var paramsRaw json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshaling params: %w", err)
		}
		paramsRaw = data
	}

	req := Request{
		Method: method,
		Params: paramsRaw,
		ID:     id,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}
	data = append(data, '\n')

	c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := c.conn.Write(data); err != nil {
		return nil, fmt.Errorf("writing request: %w", err)
	}

	c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	if !c.scanner.Scan() {
		if err := c.scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}
		return nil, fmt.Errorf("connection closed")
	}

	var resp Response
	if err := json.Unmarshal(c.scanner.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("agent error: %s", resp.Error)
	}

	return &resp, nil
}

// Ping checks if the agent is reachable.
func (c *Client) Ping() error {
	resp, err := c.Call("ping", nil)
	if err != nil {
		return err
	}
	if resp.Result != "pong" {
		return fmt.Errorf("unexpected ping response: %v", resp.Result)
	}
	return nil
}

// Heartbeat starts a background goroutine that pings the agent every interval.
// Returns a channel that receives true on success, false on failure.
func (c *Client) Heartbeat(interval time.Duration, stop <-chan struct{}) <-chan bool {
	ch := make(chan bool, 1)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				err := c.Ping()
				select {
				case ch <- (err == nil):
				default:
				}
			}
		}
	}()

	return ch
}
