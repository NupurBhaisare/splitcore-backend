package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/splitcore/backend/internal/database"
	"github.com/splitcore/backend/internal/migrations"
	"github.com/splitcore/backend/internal/routes"
)

func main() {
	// Load .env file if present
	_ = godotenv.Load()

	log.Println("Starting SplitCore Backend...")

	// Initialize database
	if err := database.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := migrations.RunAll(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Setup router
	router := routes.NewRouter()

	// Get port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15,
		WriteTimeout: 15,
		IdleTimeout:  60,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		server.Close()
	}()

	log.Printf("Server running on port %s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server stopped")
}
