package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/reelchoice/backend/internal/database"
	"github.com/reelchoice/backend/internal/party"
	"github.com/reelchoice/backend/internal/tmdb"
)

// Hub manages WebSocket connections for all parties
type Hub struct {
	// Active connections for each party (partyID -> connections)
	parties map[string]map[*websocket.Conn]bool
	mutex   sync.RWMutex

	// Redis client for state management
	redis *database.RedisClient

	// TMDB client for movie data
	tmdbClient *tmdb.Client

	// Token manager for authentication
	tokenManager *party.TokenManager

	// Party service for business logic
	partyService *party.Service

	// Channels for connection management
	register   chan *Connection
	unregister chan *Connection
	broadcast  chan *BroadcastMessage
}

// Connection represents a WebSocket connection with metadata
type Connection struct {
	Conn     *websocket.Conn
	PartyID  string
	UserID   string
	Username string
}

// BroadcastMessage represents a message to broadcast to a party
type BroadcastMessage struct {
	PartyID string
	Data    []byte
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		// In production, validate the origin properly
		return true
	},
}

// NewHub creates a new WebSocket hub
func NewHub(redisClient *database.RedisClient) *Hub {
	return &Hub{
		parties:    make(map[string]map[*websocket.Conn]bool),
		redis:      redisClient,
		register:   make(chan *Connection),
		unregister: make(chan *Connection),
		broadcast:  make(chan *BroadcastMessage),
	}
}

// SetTMDBClient sets the TMDB client for the hub
func (h *Hub) SetTMDBClient(tmdbClient *tmdb.Client) {
	h.tmdbClient = tmdbClient
}

// SetTokenManager sets the token manager for the hub
func (h *Hub) SetTokenManager(tokenManager *party.TokenManager) {
	h.tokenManager = tokenManager
}

// SetPartyService sets the party service for the hub
func (h *Hub) SetPartyService(partyService *party.Service) {
	h.partyService = partyService
}

// Run starts the hub and handles connection management
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.registerConnection(conn)

		case conn := <-h.unregister:
			h.unregisterConnection(conn)

		case message := <-h.broadcast:
			h.broadcastToParty(message.PartyID, message.Data)
		}
	}
}

// registerConnection adds a new connection to the hub
func (h *Hub) registerConnection(conn *Connection) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.parties[conn.PartyID] == nil {
		h.parties[conn.PartyID] = make(map[*websocket.Conn]bool)
	}

	h.parties[conn.PartyID][conn.Conn] = true
	log.Printf("User %s connected to party %s. Total connections: %d",
		conn.Username, conn.PartyID, len(h.parties[conn.PartyID]))
}

// unregisterConnection removes a connection from the hub
func (h *Hub) unregisterConnection(conn *Connection) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if connections, exists := h.parties[conn.PartyID]; exists {
		if connections[conn.Conn] {
			delete(connections, conn.Conn)
			conn.Conn.Close()

			// Remove party if no connections left
			if len(connections) == 0 {
				delete(h.parties, conn.PartyID)
			}

			log.Printf("User %s disconnected from party %s. Remaining connections: %d",
				conn.Username, conn.PartyID, len(connections))
		}
	}
}

// Broadcast sends a message to all connections in a party
func (h *Hub) Broadcast(partyID string, message []byte) {
	select {
	case h.broadcast <- &BroadcastMessage{PartyID: partyID, Data: message}:
	default:
		log.Printf("Broadcast channel full, dropping message for party %s", partyID)
	}
}

// broadcastToParty sends data to all connections in a specific party
func (h *Hub) broadcastToParty(partyID string, data []byte) {
	h.mutex.RLock()
	connections := h.parties[partyID]
	h.mutex.RUnlock()

	if connections == nil {
		return
	}

	for conn := range connections {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("Error writing to websocket: %v", err)
			// Connection will be cleaned up by the read pump
		}
	}
}

// ServeWS handles WebSocket connections
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	// Extract party ID from URL
	partyID := chi.URLParam(r, "partyID")
	if partyID == "" {
		http.Error(w, "Party ID is required", http.StatusBadRequest)
		return
	}

	// Extract and validate auth token
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Authentication token is required", http.StatusUnauthorized)
		return
	}

	if h.tokenManager == nil {
		http.Error(w, "Authentication not configured", http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	tokenInfo, err := h.tokenManager.ValidateToken(ctx, token)
	if err != nil {
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	// Verify token is for this party
	if tokenInfo.PartyID != partyID {
		http.Error(w, "Token not valid for this party", http.StatusForbidden)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Get or create party
	partyData, err := h.redis.GetParty(ctx, partyID)
	if err != nil {
		log.Printf("Error getting party %s: %v", partyID, err)
		conn.Close()
		return
	}

	if partyData == nil {
		log.Printf("Party %s not found", partyID)
		conn.Close()
		return
	}

	// User info comes from validated token
	userID := tokenInfo.UserID
	username := tokenInfo.Username

	// Verify user is still in the party (they might have been removed)
	if participant := partyData.GetParticipant(userID); participant == nil {
		log.Printf("User %s (ID: %s) no longer in party %s", username, userID, partyID)
		conn.Close()
		return
	}

	connection := &Connection{
		Conn:     conn,
		PartyID:  partyID,
		UserID:   userID,
		Username: username,
	}

	// Register the connection
	h.register <- connection

	// Start goroutines for this connection
	go h.writePump(connection)
	go h.readPump(connection)
}

// writePump handles sending messages to the WebSocket connection
func (h *Hub) writePump(conn *Connection) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		conn.Conn.Close()
	}()

	for {
		select {
		case <-ticker.C:
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump handles receiving messages from the WebSocket connection
func (h *Hub) readPump(conn *Connection) {
	defer func() {
		h.unregister <- conn
		// Note: We no longer remove users from party state on disconnect
		// Users remain in the party until they explicitly leave or the party expires
	}()

	conn.Conn.SetReadLimit(512)
	conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.Conn.SetPongHandler(func(string) error {
		conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages
		h.handleMessage(conn, message)
	}
}

// handleMessage processes incoming WebSocket messages
func (h *Hub) handleMessage(conn *Connection, message []byte) {
	var msg party.Message
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Error parsing message: %v", err)
		h.sendError(conn, "Invalid message format")
		return
	}

	ctx := context.Background()

	switch msg.Type {
	case party.MessageTypePing:
		h.handlePing(conn)

	case party.MessageTypeSearchMovies:
		h.handleSearchMovies(ctx, conn, &msg)

	case party.MessageTypeSuggestMovie:
		h.handleSuggestMovie(ctx, conn, &msg)

	case party.MessageTypeVoteNomination:
		h.handleVoteNomination(ctx, conn, &msg)

	case party.MessageTypeFinalizeNominations:
		h.handleFinalizeNominations(ctx, conn, &msg)

	case party.MessageTypeSubmitRanking:
		h.handleSubmitRanking(ctx, conn, &msg)

	default:
		log.Printf("Unknown message type: %s", msg.Type)
		h.sendError(conn, "Unknown message type")
	}
}

// handlePing responds to ping messages
func (h *Hub) handlePing(conn *Connection) {
	response := map[string]interface{}{
		"type":      party.MessageTypePong,
		"timestamp": time.Now().Unix(),
	}
	responseData, _ := json.Marshal(response)
	conn.Conn.WriteMessage(websocket.TextMessage, responseData)
}

// handleSearchMovies handles movie search requests
func (h *Hub) handleSearchMovies(ctx context.Context, conn *Connection, msg *party.Message) {
	if h.partyService == nil {
		h.sendError(conn, "Party service not available")
		return
	}

	var payload party.SearchMoviesPayload
	if err := msg.ParsePayload(&payload); err != nil {
		h.sendError(conn, "Invalid search payload")
		return
	}

	movies, err := h.partyService.SearchMovies(ctx, payload.Query)
	if err != nil {
		log.Printf("Error searching movies: %v", err)
		h.sendError(conn, "Movie search failed")
		return
	}

	// Send search results back to the requesting client
	response := party.SearchResultsPayload{
		Query:  payload.Query,
		Movies: movies,
	}

	responseMsg, err := party.CreateMessage(party.MessageTypeSearchResults, response)
	if err != nil {
		h.sendError(conn, "Failed to create response")
		return
	}

	responseData, _ := json.Marshal(responseMsg)
	conn.Conn.WriteMessage(websocket.TextMessage, responseData)
}

// handleSuggestMovie handles movie suggestion messages
func (h *Hub) handleSuggestMovie(ctx context.Context, conn *Connection, msg *party.Message) {
	if h.partyService == nil {
		h.sendError(conn, "Party service not available")
		return
	}

	var payload party.SuggestMoviePayload
	if err := msg.ParsePayload(&payload); err != nil {
		h.sendError(conn, "Invalid suggestion payload")
		return
	}

	// Use the party service to handle the suggestion
	updatedParty, err := h.partyService.SuggestMovie(ctx, conn.PartyID, conn.UserID, payload.TMDBID)
	if err != nil {
		log.Printf("Error suggesting movie: %v", err)
		h.sendError(conn, err.Error())
		return
	}

	// Broadcast updated party state
	h.broadcastPartyState(updatedParty)
}

// handleVoteNomination handles nomination voting
func (h *Hub) handleVoteNomination(ctx context.Context, conn *Connection, msg *party.Message) {
	if h.partyService == nil {
		h.sendError(conn, "Party service not available")
		return
	}

	var payload party.VotePayload
	if err := msg.ParsePayload(&payload); err != nil {
		h.sendError(conn, "Invalid vote payload")
		return
	}

	// Use the party service to handle the vote
	updatedParty, err := h.partyService.VoteNomination(ctx, conn.PartyID, conn.UserID, payload.Vote)
	if err != nil {
		log.Printf("Error voting on nomination: %v", err)
		h.sendError(conn, err.Error())
		return
	}

	// Broadcast updated party state
	h.broadcastPartyState(updatedParty)
}

// handleFinalizeNominations handles finalization of nominations (host only)
func (h *Hub) handleFinalizeNominations(ctx context.Context, conn *Connection, msg *party.Message) {
	if h.partyService == nil {
		h.sendError(conn, "Party service not available")
		return
	}

	// Use the party service to finalize nominations
	updatedParty, err := h.partyService.FinalizeNominations(ctx, conn.PartyID, conn.UserID)
	if err != nil {
		log.Printf("Error finalizing nominations: %v", err)
		h.sendError(conn, err.Error())
		return
	}

	// Broadcast updated party state
	h.broadcastPartyState(updatedParty)
}

// handleSubmitRanking handles ranking submission messages
func (h *Hub) handleSubmitRanking(ctx context.Context, conn *Connection, msg *party.Message) {
	if h.partyService == nil {
		h.sendError(conn, "Party service not available")
		return
	}

	var payload party.SubmitRankingPayload
	if err := msg.ParsePayload(&payload); err != nil {
		h.sendError(conn, "Invalid ranking payload")
		return
	}

	// Use the party service to handle the ranking submission
	updatedParty, err := h.partyService.SubmitRanking(ctx, conn.PartyID, conn.UserID, payload.Ranks)
	if err != nil {
		log.Printf("Error submitting ranking: %v", err)
		h.sendError(conn, err.Error())
		return
	}

	// Log if party is completed
	if updatedParty.Phase == party.PhaseFinished && updatedParty.Winner != nil {
		log.Printf("Party %s completed: Winner is %s", updatedParty.ID, updatedParty.Winner.Title)
	}

	// Broadcast updated party state
	h.broadcastPartyState(updatedParty)
}

// sendError sends an error message to a specific connection
func (h *Hub) sendError(conn *Connection, errorMsg string) {
	errorPayload := party.ErrorPayload{Error: errorMsg}
	response, err := party.CreateMessage("error", errorPayload)
	if err != nil {
		log.Printf("Error creating error message: %v", err)
		return
	}

	responseData, _ := json.Marshal(response)
	conn.Conn.WriteMessage(websocket.TextMessage, responseData)
}

// broadcastPartyState sends the current party state to all connections
func (h *Hub) broadcastPartyState(partyData *party.Party) {
	payload := party.PartyUpdatePayload{Party: partyData}
	msg, err := party.CreateMessage(party.MessageTypePartyUpdate, payload)
	if err != nil {
		log.Printf("Error creating party update message: %v", err)
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling party state: %v", err)
		return
	}

	h.Broadcast(partyData.ID, data)
}
