package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pinkpanel/pinkpanel/internal/agent"
)

var version = "0.6.1-alpha"

func main() {
	socket := flag.String("socket", "", "Unix socket path")
	flag.Parse()

	socketPath := *socket
	if socketPath == "" {
		socketPath = os.Getenv("PINKPANEL_AGENT_SOCKET")
	}
	if socketPath == "" {
		socketPath = "/tmp/pinkpanel-agent/agent.sock"
	}

	server := agent.NewServer(socketPath)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-quit
		fmt.Println("\nAgent shutting down...")
		server.Stop()
		os.Exit(0)
	}()

	fmt.Printf("PinkPanel Agent %s\n", version)
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Agent failed: %v\n", err)
		os.Exit(1)
	}
}
