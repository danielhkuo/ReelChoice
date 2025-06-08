// backend/main.go
package main

import (
	"log"
	"net/http"
	"reelchoice/backend/api" // Import our local api package
	"reelchoice/backend/websocket"

	// Import our WebSocket package
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// RequestLogger is a middleware that logs detailed information about each incoming request.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("--- Go Backend: Incoming Request ---")
		log.Printf("Request URL: %s %s", r.Method, r.URL.String())
		log.Printf("Host: %s", r.Host)
		log.Printf("User-Agent: %s", r.UserAgent())
		log.Printf("Remote Address: %s", r.RemoteAddr)
		next.ServeHTTP(w, r)
		log.Println("--- Go Backend: Request Handled ---")
	})
}

func main() {
	// Create a new Chi router
	r := chi.NewRouter()

	// Use some helpful middleware
	r.Use(middleware.Logger)    // Logs requests to the console
	r.Use(middleware.Recoverer) // Recovers from panics without crashing
	r.Use(RequestLogger)        // Add our custom detailed logger

	// Create and start WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Define our API routes
	r.Post("/api/party", api.CreatePartyHandler)
	r.Get("/api/party/{id}", api.GetPartyHandler)
	r.Post("/api/party/{id}/join", api.JoinPartyHandler)

	// WebSocket route
	r.Get("/ws", hub.ServeWS)

	// Add a simple test endpoint
	r.Get("/api/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok", "message": "Backend is running!"}`))
	})

	// Start the server
	port := "8081"
	log.Printf("Backend server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
