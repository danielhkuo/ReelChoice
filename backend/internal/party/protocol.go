package party

import "encoding/json"

// Message represents a WebSocket message
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Message types
const (
	MessageTypePing                = "ping"
	MessageTypePong                = "pong"
	MessageTypePartyUpdate         = "party_update"
	MessageTypeUserJoined          = "user_joined"
	MessageTypeUserLeft            = "user_left"
	MessageTypeSuggestMovie        = "suggest_movie"
	MessageTypeVoteNomination      = "vote_nomination"
	MessageTypeFinalizeNominations = "finalize_nominations"
	MessageTypeSubmitRanking       = "submit_ranking"
	MessageTypeSearchMovies        = "search_movies"
	MessageTypeSearchResults       = "search_results"
)

// SuggestMoviePayload represents a movie suggestion payload
type SuggestMoviePayload struct {
	TMDBID string `json:"tmdb_id"`
}

// VotePayload represents a nomination vote payload
type VotePayload struct {
	Vote string `json:"vote"` // "yay" or "nay"
}

// SearchMoviesPayload represents a movie search request
type SearchMoviesPayload struct {
	Query string `json:"query"`
}

// SearchResultsPayload represents movie search results
type SearchResultsPayload struct {
	Query  string  `json:"query"`
	Movies []Movie `json:"movies"`
}

// SubmitRankingPayload represents a ranking submission
type SubmitRankingPayload struct {
	Ranks []string `json:"ranks"` // Array of movie IDs in ranked order
}

// PartyUpdatePayload represents a party state update
type PartyUpdatePayload struct {
	Party *Party `json:"party"`
}

// ErrorPayload represents an error message
type ErrorPayload struct {
	Error string `json:"error"`
}

// CreateMessage creates a new message with the given type and payload
func CreateMessage(msgType string, payload interface{}) (*Message, error) {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &Message{
		Type:    msgType,
		Payload: payloadData,
	}, nil
}

// ParsePayload parses the message payload into the given interface
func (m *Message) ParsePayload(v interface{}) error {
	return json.Unmarshal(m.Payload, v)
}
