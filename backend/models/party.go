// backend/models/party.go
package models

import "time"

// PartyState represents the current state of the party
type PartyState string

const (
	StateLobby      PartyState = "lobby"
	StateNomination PartyState = "nomination"
	StateVoting     PartyState = "voting"
	StateCompleted  PartyState = "completed"
)

// Participant represents a user in a party
type Participant struct {
	ID       string    `json:"id"`
	Username string    `json:"username"`
	IsHost   bool      `json:"isHost"`
	JoinedAt time.Time `json:"joinedAt"`
}

// Party represents a movie selection party session.
type Party struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	HostID       string        `json:"hostId"`
	State        PartyState    `json:"state"`
	Participants []Participant `json:"participants"`
	CreatedAt    time.Time     `json:"createdAt"`
	// We will add more fields later, like Nominations, Votes, etc.
}

// AddParticipant adds a new participant to the party
func (p *Party) AddParticipant(userID, username string) {
	participant := Participant{
		ID:       userID,
		Username: username,
		IsHost:   userID == p.HostID,
		JoinedAt: time.Now(),
	}
	p.Participants = append(p.Participants, participant)
}

// RemoveParticipant removes a participant from the party
func (p *Party) RemoveParticipant(userID string) {
	for i, participant := range p.Participants {
		if participant.ID == userID {
			p.Participants = append(p.Participants[:i], p.Participants[i+1:]...)
			break
		}
	}
}

// GetParticipant returns a participant by ID
func (p *Party) GetParticipant(userID string) *Participant {
	for _, participant := range p.Participants {
		if participant.ID == userID {
			return &participant
		}
	}
	return nil
}

// IsHost checks if a user is the host of the party
func (p *Party) IsHost(userID string) bool {
	return p.HostID == userID
}
