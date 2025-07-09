package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"iaros/api_gateway/src/config"
	"iaros/api_gateway/src/gateway"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create API Gateway instance
	gw, err := gateway.NewGateway(cfg)
	if err != nil {
		log.Fatalf("Failed to create API Gateway: %v", err)
	}

	// Start the gateway in a goroutine
	go func() {
		log.Printf("Starting API Gateway on port %d", cfg.Server.Port)
		if err := gw.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start API Gateway: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down API Gateway...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown the gateway
	if err := gw.Shutdown(ctx); err != nil {
		log.Fatalf("API Gateway forced to shutdown: %v", err)
	}

	log.Println("API Gateway stopped")
} 