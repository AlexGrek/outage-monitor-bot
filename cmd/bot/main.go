package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"tg-monitor-bot/internal/appmanager"
	"tg-monitor-bot/internal/storage"
)

func main() {
	log.Println("🤖 Starting Outage Monitor Bot with AppManager...")

	// Initialize database
	db, err := storage.NewBoltDB("data/state.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create AppManager
	manager := appmanager.New(db)

	// Start AppManager (ConfigManager + Echo API + Bot)
	if err := manager.Start(); err != nil {
		log.Fatalf("Failed to start AppManager: %v", err)
	}

	log.Println("✅ Application started successfully")
	log.Println("📡 Bot is running and monitoring sources")
	log.Println("🌐 API server is available for config management")
	log.Println("Press Ctrl+C to stop")

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutdown signal received...")
	manager.Shutdown()
	log.Println("✅ Shutdown complete")
}
