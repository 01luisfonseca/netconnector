package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"netconnector/internal/client"
	"netconnector/pkg/logger"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		logger.Debug("No .env file found, relying on environment variables")
	}

	// Simple config via ENV
	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = "localhost:50051" // Default VPS address
	}

	clientID := os.Getenv("CLIENT_ID")
	if clientID == "" {
		logger.Error("CLIENT_ID is required to start the agent")
		os.Exit(1)
	}

	localAppURL := os.Getenv("LOCAL_APP_URL")
	if localAppURL == "" {
		localAppURL = "http://localhost:3000" // Default targeted local app
	}

	logger.Info("Starting Netconnector Local Agent",
		"server_addr", serverAddr,
		"client_id", clientID,
		"local_app", localAppURL)

	// Create dependencies
	executor := client.NewExecutor(localAppURL)
	agent := client.NewAgent(serverAddr, clientID, executor)

	// Context with graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Capture interrupt signals for graceful stop
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		logger.Info("Interrupt received, shutting down agent...")
		cancel()
	}()

	// Start Agent blocking loop
	if err := agent.Start(ctx); err != nil {
		logger.Error("Agent stopped with error", "err", err)
		os.Exit(1)
	}

	logger.Info("Agent exited securely.")
}
