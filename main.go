package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"jmg/internal/config"
	"jmg/internal/database"
	"jmg/internal/server"
	"jmg/internal/storage"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg.Storage.DataDir)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize disk storage
	disk, err := storage.NewDisk(cfg.Storage.DataDir)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize thumbnail generator
	thumbGen := storage.NewThumbnailGenerator(cfg.Storage.DataDir, cfg.Thumbnail)

	// Create and start server
	srv := server.New(cfg, db, disk, thumbGen)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		log.Printf("🖼️  ImgHost starting on http://%s", addr)
		if cfg.Server.BaseURL != "" {
			log.Printf("📡 External URL: %s", cfg.Server.BaseURL)
		}
		if err := srv.Start(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down...")
	srv.Shutdown()
}
