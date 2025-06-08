package party

import "time"

// Movie represents a movie with TMDB data
type Movie struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Year       string `json:"year"`
	PosterPath string `json:"poster_path"`
}

// Participant represents a user in a party
type Participant struct {
	ID       string `json:"id"` // Unique ID for this participant
	Username string `json:"username"`
	IsHost   bool   `json:"is_host"`
}

// NominationVote represents a movie being voted on for nomination
type NominationVote struct {
	Movie  Movie             `json:"movie"`
	Voters map[string]string `json:"voters"` // Map participant ID to their vote ("yay" or "nay")
}

// Party represents the complete state of a party session
type Party struct {
	ID           string                  `json:"id"`
	Name         string                  `json:"name"`
	Participants map[string]*Participant `json:"participants"` // Map of participant ID to participant
	Phase        string                  `json:"phase"`        // "lobby", "nominating", "ranking", "finished"
	CreatedAt    time.Time               `json:"created_at"`

	// Nomination phase fields
	CurrentNomination *NominationVote `json:"current_nomination"`
	NominationPool    []Movie         `json:"nomination_pool"`

	// Ranking phase fields
	Submissions map[string][]string `json:"submissions"` // Map participant ID to their ranked list of Movie IDs
	Winner      *Movie              `json:"winner"`
}

// Phase constants
const (
	PhaseLobby      = "lobby"
	PhaseNominating = "nominating"
	PhaseRanking    = "ranking"
	PhaseFinished   = "finished"
)

// AddParticipant adds a new participant to the party
func (p *Party) AddParticipant(userID, username string, isHost bool) {
	if p.Participants == nil {
		p.Participants = make(map[string]*Participant)
	}

	p.Participants[userID] = &Participant{
		ID:       userID,
		Username: username,
		IsHost:   isHost,
	}
}

// RemoveParticipant removes a participant from the party
func (p *Party) RemoveParticipant(userID string) {
	delete(p.Participants, userID)
}

// GetParticipant returns a participant by ID
func (p *Party) GetParticipant(userID string) *Participant {
	return p.Participants[userID]
}

// IsHost checks if a user is the host of the party
func (p *Party) IsHost(userID string) bool {
	participant := p.GetParticipant(userID)
	return participant != nil && participant.IsHost
}

// GetHost returns the host participant
func (p *Party) GetHost() *Participant {
	for _, participant := range p.Participants {
		if participant.IsHost {
			return participant
		}
	}
	return nil
}

// ParticipantCount returns the number of participants in the party
func (p *Party) ParticipantCount() int {
	return len(p.Participants)
}
