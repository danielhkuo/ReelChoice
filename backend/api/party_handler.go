// backend/api/party_handler.go
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"reelchoice/backend/models" // Import our local models package
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teris-io/shortid"
)

// Note: This in-memory map is a temporary placeholder.
// It is NOT thread-safe and will be replaced by Redis later.
var parties = make(map[string]*models.Party)

// CreatePartyHandler handles the creation of a new party.
func CreatePartyHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Decode the incoming JSON request
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

	// 2. Create a new Party object
	newID, err := shortid.Generate()
	if err != nil {
		http.Error(w, "Failed to generate party ID", http.StatusInternalServerError)
		return
	}

	// Generate a host ID (in real implementation, this would come from authentication)
	hostID, err := shortid.Generate()
	if err != nil {
		http.Error(w, "Failed to generate host ID", http.StatusInternalServerError)
		return
	}

	party := &models.Party{
		ID:           newID,
		Name:         req.Name,
		HostID:       hostID,
		State:        models.StateLobby,
		Participants: []models.Participant{},
		CreatedAt:    time.Now(),
	}

	// 3. Store the party (in our temporary map)
	parties[party.ID] = party
	log.Printf("Party created: %s (ID: %s)", party.Name, party.ID)

	// 4. Respond with the newly created party
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 Created
	json.NewEncoder(w).Encode(party)
}

// GetPartyHandler handles getting party information by ID
func GetPartyHandler(w http.ResponseWriter, r *http.Request) {
	partyID := chi.URLParam(r, "id")
	if partyID == "" {
		http.Error(w, "Party ID is required", http.StatusBadRequest)
		return
	}

	party, exists := parties[partyID]
	if !exists {
		http.Error(w, "Party not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(party)
}

// JoinPartyHandler handles a user joining a party
func JoinPartyHandler(w http.ResponseWriter, r *http.Request) {
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

	party, exists := parties[partyID]
	if !exists {
		http.Error(w, "Party not found", http.StatusNotFound)
		return
	}

	// Generate a user ID (in real implementation, this would come from authentication)
	userID, err := shortid.Generate()
	if err != nil {
		http.Error(w, "Failed to generate user ID", http.StatusInternalServerError)
		return
	}

	// Check if username is already taken in this party
	for _, participant := range party.Participants {
		if participant.Username == req.Username {
			http.Error(w, "Username already taken in this party", http.StatusConflict)
			return
		}
	}

	// Add participant to party
	party.AddParticipant(userID, req.Username)
	log.Printf("User %s (%s) joined party %s", req.Username, userID, partyID)

	// Return the participant info including generated userID
	participant := party.GetParticipant(userID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"participant": participant,
		"party":       party,
	})
}
