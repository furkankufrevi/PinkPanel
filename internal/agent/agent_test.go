package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCommandRegistry(t *testing.T) {
	reg := NewCommandRegistry()

	// Test ping
	result, err := reg.Execute("ping", nil)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if result != "pong" {
		t.Errorf("Expected pong, got %v", result)
	}

	// Test unknown command
	_, err = reg.Execute("unknown_command", nil)
	if err == nil {
		t.Error("Expected error for unknown command")
	}
}

func TestServiceStatusAllowlist(t *testing.T) {
	reg := NewCommandRegistry()

	// Allowed service
	params, _ := json.Marshal(serviceStatusParams{Service: "nginx"})
	result, err := reg.Execute("service_status", params)
	if err != nil {
		t.Fatalf("service_status for nginx failed: %v", err)
	}
	status, ok := result.(serviceStatusResult)
	if !ok {
		t.Fatalf("Expected serviceStatusResult, got %T", result)
	}
	if status.Service != "nginx" {
		t.Errorf("Expected service nginx, got %s", status.Service)
	}

	// Disallowed service
	params, _ = json.Marshal(serviceStatusParams{Service: "evil-service"})
	_, err = reg.Execute("service_status", params)
	if err == nil {
		t.Error("Expected error for disallowed service")
	}

	// Shell injection attempt
	params, _ = json.Marshal(serviceStatusParams{Service: "nginx; rm -rf /"})
	_, err = reg.Execute("service_status", params)
	if err == nil {
		t.Error("Expected error for shell injection")
	}
}

func TestSystemInfo(t *testing.T) {
	reg := NewCommandRegistry()

	result, err := reg.Execute("system_info", nil)
	if err != nil {
		t.Fatalf("system_info failed: %v", err)
	}
	info, ok := result.(systemInfoResult)
	if !ok {
		t.Fatalf("Expected systemInfoResult, got %T", result)
	}
	if info.OS == "" {
		t.Error("OS should not be empty")
	}
	if info.Arch == "" {
		t.Error("Arch should not be empty")
	}
}

func TestServerClientIntegration(t *testing.T) {
	// Create a temp socket
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "agent.sock")

	// Start server
	server := NewServer(socketPath)
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Verify socket exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Fatal("Socket file not created")
	}

	// Connect client
	client := NewClient(socketPath)
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Test ping
	if err := client.Ping(); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Test service_status
	resp, err := client.Call("service_status", serviceStatusParams{Service: "nginx"})
	if err != nil {
		t.Fatalf("service_status call failed: %v", err)
	}
	if resp.Result == nil {
		t.Error("Expected non-nil result")
	}

	// Test system_info
	resp, err = client.Call("system_info", nil)
	if err != nil {
		t.Fatalf("system_info call failed: %v", err)
	}
	if resp.Result == nil {
		t.Error("Expected non-nil result")
	}

	// Cleanup
	server.Stop()
}
