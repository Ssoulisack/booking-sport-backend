package main

import (
	"backend/cmd/application"
	"backend/internal/adapters/primary/http"

	"backend/internal/config"
	"backend/pkg/logger"
	"log"
	"time"
)

func main() {
	// Initialize logger
	logger.NewLogger()
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	// Initialize HTTP server
	serverConfig := http.ServerConfig{
		Port:         cfg.Server.Port,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}
	server := http.NewServer(
		serverConfig,
	)

	repositories := application.SetupRepositories(cfg)
	defer application.CloseRepositories(repositories)
	services := application.SetupServices(repositories)
	handlers := application.SetupHandlers(services)
	application.SetupRoutes(server.App(), handlers)

	// Start HTTP server
	log.Println("🌟 Server starting on port:", cfg.Server.Port, "🚀")
	if err := server.Start(); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}
