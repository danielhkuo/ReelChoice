package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/reelchoice/backend/internal/config"
	"github.com/reelchoice/backend/internal/database"
	"github.com/reelchoice/backend/internal/party"
	"github.com/reelchoice/backend/internal/tmdb"
	"github.com/reelchoice/backend/internal/websocket"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handlers contains all the HTTP handlers and their dependencies
type Handlers struct {
	redis        *database.RedisClient
	hub          *websocket.Hub
	config       *config.Config
	tmdbClient   *tmdb.Client
	tokenManager *party.TokenManager
	partyService *party.Service
}

// NewHandlers creates a new Handlers instance with dependencies
func NewHandlers(redis *database.RedisClient, hub *websocket.Hub, cfg *config.Config) *Handlers {
	// Create TMDB client
	tmdbClient := tmdb.NewClient(cfg.TMDBApiKey, redis)

	// Create token manager with Redis backend
	tokenManager := party.NewTokenManager(redis)

	// Create party service
	partyService := party.NewService(redis, tmdbClient)

	// Set TMDB client in the hub
	hub.SetTMDBClient(tmdbClient)
	hub.SetTokenManager(tokenManager)
	hub.SetPartyService(partyService)

	return &Handlers{
		redis:        redis,
		hub:          hub,
		config:       cfg,
		tmdbClient:   tmdbClient,
		tokenManager: tokenManager,
		partyService: partyService,
	}
}

// CreateParty handles POST /api/party
func (h *Handlers) CreateParty(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Party name cannot be empty", http.StatusBadRequest)
		return
	}

	// Generate party ID
	partyID := uuid.New().String()

	// Generate host ID
	hostID := uuid.New().String()

	// Create new party
	newParty := &party.Party{
		ID:           partyID,
		Name:         req.Name,
		Participants: make(map[string]*party.Participant),
		Phase:        party.PhaseLobby,
		CreatedAt:    time.Now(),
	}

	// Add creator as host
	newParty.AddParticipant(hostID, "Host", true)

	// Save to Redis
	ctx := context.Background()
	if err := h.redis.SaveParty(ctx, newParty); err != nil {
		log.Printf("Error saving party to Redis: %v", err)
		http.Error(w, "Failed to create party", http.StatusInternalServerError)
		return
	}

	log.Printf("Party created: %s (ID: %s)", newParty.Name, newParty.ID)

	// Create authentication token for the host
	authToken, err := h.tokenManager.CreateToken(ctx, partyID, hostID, "Host", true)
	if err != nil {
		log.Printf("Error creating host auth token: %v", err)
		http.Error(w, "Failed to create authentication token", http.StatusInternalServerError)
		return
	}

	// Return party info with auth token
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"party_id":   partyID,
		"host_id":    hostID,
		"party":      newParty,
		"auth_token": authToken.Token,
	})
}

// GetParty handles GET /api/party/{id}
func (h *Handlers) GetParty(w http.ResponseWriter, r *http.Request) {
	partyID := chi.URLParam(r, "id")
	if partyID == "" {
		http.Error(w, "Party ID is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	party, err := h.redis.GetParty(ctx, partyID)
	if err != nil {
		log.Printf("Error getting party %s: %v", partyID, err)
		http.Error(w, "Failed to get party", http.StatusInternalServerError)
		return
	}

	if party == nil {
		http.Error(w, "Party not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(party)
}

// JoinParty handles POST /api/party/{id}/join
func (h *Handlers) JoinParty(w http.ResponseWriter, r *http.Request) {
	partyID := chi.URLParam(r, "id")
	if partyID == "" {
		http.Error(w, "Party ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		http.Error(w, "Username cannot be empty", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	party, err := h.redis.GetParty(ctx, partyID)
	if err != nil {
		log.Printf("Error getting party %s: %v", partyID, err)
		http.Error(w, "Failed to get party", http.StatusInternalServerError)
		return
	}

	if party == nil {
		http.Error(w, "Party not found", http.StatusNotFound)
		return
	}

	// Check if username is already taken
	for _, participant := range party.Participants {
		if participant.Username == req.Username {
			http.Error(w, "Username already taken in this party", http.StatusConflict)
			return
		}
	}

	// Generate user ID
	userID := uuid.New().String()

	// Add participant to party
	party.AddParticipant(userID, req.Username, false)

	// Save updated party
	if err := h.redis.SaveParty(ctx, party); err != nil {
		log.Printf("Error saving party after join: %v", err)
		http.Error(w, "Failed to join party", http.StatusInternalServerError)
		return
	}

	log.Printf("User %s (%s) joined party %s", req.Username, userID, partyID)

	// Create authentication token
	authToken, err := h.tokenManager.CreateToken(ctx, partyID, userID, req.Username, false)
	if err != nil {
		log.Printf("Error creating auth token: %v", err)
		http.Error(w, "Failed to create authentication token", http.StatusInternalServerError)
		return
	}

	// Broadcast party update to all connected clients
	h.broadcastPartyUpdate(party)

	// Return response with auth token
	participant := party.GetParticipant(userID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":     userID,
		"participant": participant,
		"party":       party,
		"auth_token":  authToken.Token,
	})
}

// StartNomination handles POST /api/party/{id}/start-nomination (host only)
func (h *Handlers) StartNomination(w http.ResponseWriter, r *http.Request) {
	partyID := chi.URLParam(r, "id")
	if partyID == "" {
		http.Error(w, "Party ID is required", http.StatusBadRequest)
		return
	}

	// Extract and validate auth token
	authToken := h.extractAuthToken(r)
	if authToken == "" {
		http.Error(w, "Authorization token required", http.StatusUnauthorized)
		return
	}

	ctx := context.Background()
	tokenInfo, err := h.tokenManager.ValidateToken(ctx, authToken)
	if err != nil {
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	// Verify token is for this party
	if tokenInfo.PartyID != partyID {
		http.Error(w, "Token not valid for this party", http.StatusForbidden)
		return
	}

	// Verify user is host
	if !tokenInfo.IsHost {
		http.Error(w, "Only the host can start nomination phase", http.StatusForbidden)
		return
	}
	partyData, err := h.redis.GetParty(ctx, partyID)
	if err != nil {
		log.Printf("Error getting party %s: %v", partyID, err)
		http.Error(w, "Failed to get party", http.StatusInternalServerError)
		return
	}

	if partyData == nil {
		http.Error(w, "Party not found", http.StatusNotFound)
		return
	}

	// User is already verified as host by token validation above

	// Check if party is in lobby phase
	if partyData.Phase != party.PhaseLobby {
		http.Error(w, "Party must be in lobby phase to start nominations", http.StatusBadRequest)
		return
	}

	// Change phase to nominating
	partyData.Phase = party.PhaseNominating

	// Initialize nomination fields
	partyData.CurrentNomination = nil
	partyData.NominationPool = make([]party.Movie, 0)

	// Save updated party
	if err := h.redis.SaveParty(ctx, partyData); err != nil {
		log.Printf("Error saving party after starting nomination: %v", err)
		http.Error(w, "Failed to start nomination phase", http.StatusInternalServerError)
		return
	}

	log.Printf("Nomination phase started for party %s", partyID)

	// Broadcast party update
	h.broadcastPartyUpdate(partyData)

	// Return updated party
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(partyData)
}

// SearchMovies handles GET /api/movies/search
func (h *Handlers) SearchMovies(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Search query is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	movies, err := h.tmdbClient.SearchMovies(ctx, query)
	if err != nil {
		log.Printf("Error searching movies: %v", err)
		http.Error(w, "Movie search failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":  query,
		"movies": movies,
	})
}

// HealthCheck handles GET /api/health
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"message":   "ReelChoice backend is running",
		"timestamp": time.Now().Unix(),
	})
}

// broadcastPartyUpdate sends party updates to all connected WebSocket clients
func (h *Handlers) broadcastPartyUpdate(partyData *party.Party) {
	payload := party.PartyUpdatePayload{Party: partyData}
	msg, err := party.CreateMessage(party.MessageTypePartyUpdate, payload)
	if err != nil {
		log.Printf("Error creating party update message: %v", err)
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling party update: %v", err)
		return
	}

	h.hub.Broadcast(partyData.ID, data)
}

// extractAuthToken extracts the bearer token from the Authorization header
func (h *Handlers) extractAuthToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Check for "Bearer " prefix
	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		return ""
	}

	return authHeader[len(bearerPrefix):]
}
