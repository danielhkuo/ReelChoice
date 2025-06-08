package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/reelchoice/backend/internal/api"
	"github.com/reelchoice/backend/internal/config"
	"github.com/reelchoice/backend/internal/database"
	"github.com/reelchoice/backend/internal/websocket"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()
	log.Printf("Starting ReelChoice backend server on port %s", cfg.Port)

	// Initialize Redis client
	redisClient, err := database.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("Connected to Redis successfully")

	// Initialize PostgreSQL client
	pgClient, err := database.NewPostgresClient(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgClient.Close()
	log.Println("Connected to PostgreSQL successfully")

	// Create WebSocket hub
	hub := websocket.NewHub(redisClient)
	go hub.Run()
	log.Println("WebSocket hub started")

	// Set up Chi router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// CORS middleware for development
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")

			if r.Method == "OPTIONS" {
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Create API handlers with dependencies
	apiHandlers := api.NewHandlers(redisClient, hub, cfg)

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Post("/party", apiHandlers.CreateParty)
		r.Get("/party/{id}", apiHandlers.GetParty)
		r.Post("/party/{id}/join", apiHandlers.JoinParty)
		r.Post("/party/{id}/start-nomination", apiHandlers.StartNomination)
		r.Get("/movies/search", apiHandlers.SearchMovies)
		r.Get("/health", apiHandlers.HealthCheck)
	})

	// WebSocket route
	r.Get("/ws/party/{partyID}", hub.ServeWS)

	// Create HTTP server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server shutdown complete")
}
