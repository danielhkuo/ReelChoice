package party

import (
	"context"
	"fmt"
	"log"
	"time"
)

// RedisStore interface for party operations
type RedisStore interface {
	GetParty(ctx context.Context, partyID string) (*Party, error)
	SaveParty(ctx context.Context, party *Party) error
	DeleteParty(ctx context.Context, partyID string) error
	AcquireLock(ctx context.Context, partyID string, lockDuration time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, partyID string) error
	RedisTokenStore // Embed the token store interface
}

// TMDBClient interface for movie operations
type TMDBClient interface {
	SearchMovies(ctx context.Context, query string) ([]Movie, error)
	GetMovieDetails(ctx context.Context, tmdbID string) (*Movie, error)
}

// Service handles all party business logic
type Service struct {
	redis RedisStore
	tmdb  TMDBClient
}

// NewService creates a new party service
func NewService(redis RedisStore, tmdb TMDBClient) *Service {
	return &Service{
		redis: redis,
		tmdb:  tmdb,
	}
}

// WithLock executes a function while holding a distributed lock for the party
func (s *Service) WithLock(ctx context.Context, partyID string, fn func(ctx context.Context) error) error {
	const lockDuration = 5 * time.Second

	// Attempt to acquire lock
	acquired, err := s.redis.AcquireLock(ctx, partyID, lockDuration)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !acquired {
		return fmt.Errorf("party is currently being modified by another request")
	}

	// Ensure lock is always released
	defer func() {
		if releaseErr := s.redis.ReleaseLock(ctx, partyID); releaseErr != nil {
			log.Printf("Failed to release lock for party %s: %v", partyID, releaseErr)
		}
	}()

	return fn(ctx)
}

// SearchMovies searches for movies using TMDB API
func (s *Service) SearchMovies(ctx context.Context, query string) ([]Movie, error) {
	return s.tmdb.SearchMovies(ctx, query)
}

// SuggestMovie adds a movie nomination to a party
func (s *Service) SuggestMovie(ctx context.Context, partyID, userID, tmdbID string) (*Party, error) {
	var updatedParty *Party

	err := s.WithLock(ctx, partyID, func(ctx context.Context) error {
		// Get current party state
		party, err := s.redis.GetParty(ctx, partyID)
		if err != nil {
			return fmt.Errorf("failed to get party: %w", err)
		}
		if party == nil {
			return fmt.Errorf("party not found")
		}

		// Validate party phase
		if party.Phase != PhaseNominating {
			return fmt.Errorf("nominations are not open")
		}

		// Check if there's already a nomination in progress
		if party.CurrentNomination != nil {
			return fmt.Errorf("another nomination is already in progress")
		}

		// Get movie details from TMDB
		movie, err := s.tmdb.GetMovieDetails(ctx, tmdbID)
		if err != nil {
			return fmt.Errorf("failed to get movie details: %w", err)
		}
		if movie == nil {
			return fmt.Errorf("movie not found")
		}

		// Create new nomination
		party.CurrentNomination = &NominationVote{
			Movie:  *movie,
			Voters: make(map[string]string),
		}

		// Save updated party
		if err := s.redis.SaveParty(ctx, party); err != nil {
			return fmt.Errorf("failed to save party: %w", err)
		}

		updatedParty = party
		return nil
	})

	return updatedParty, err
}

// VoteNomination records a vote for a movie nomination
func (s *Service) VoteNomination(ctx context.Context, partyID, userID, vote string) (*Party, error) {
	var updatedParty *Party

	err := s.WithLock(ctx, partyID, func(ctx context.Context) error {
		// Get current party state
		party, err := s.redis.GetParty(ctx, partyID)
		if err != nil {
			return fmt.Errorf("failed to get party: %w", err)
		}
		if party == nil {
			return fmt.Errorf("party not found")
		}

		// Check if there's a nomination in progress
		if party.CurrentNomination == nil {
			return fmt.Errorf("no nomination in progress")
		}

		if vote != "yay" && vote != "nay" {
			return fmt.Errorf("vote must be 'yay' or 'nay'")
		}

		// Record the vote
		party.CurrentNomination.Voters[userID] = vote

		// Check if voting is complete (all participants have voted)
		totalParticipants := len(party.Participants)
		totalVotes := len(party.CurrentNomination.Voters)

		if totalVotes >= totalParticipants {
			// Count votes
			yayVotes := 0
			for _, vote := range party.CurrentNomination.Voters {
				if vote == "yay" {
					yayVotes++
				}
			}

			// Need majority to pass
			if yayVotes > totalParticipants/2 {
				// Add to nomination pool
				if party.NominationPool == nil {
					party.NominationPool = make([]Movie, 0)
				}
				party.NominationPool = append(party.NominationPool, party.CurrentNomination.Movie)
			}

			// Clear current nomination
			party.CurrentNomination = nil
		}

		// Save updated party
		if err := s.redis.SaveParty(ctx, party); err != nil {
			return fmt.Errorf("failed to save party: %w", err)
		}

		updatedParty = party
		return nil
	})

	return updatedParty, err
}

// FinalizeNominations moves party from nominating to ranking phase
func (s *Service) FinalizeNominations(ctx context.Context, partyID, hostID string) (*Party, error) {
	var updatedParty *Party

	err := s.WithLock(ctx, partyID, func(ctx context.Context) error {
		// Get current party state
		party, err := s.redis.GetParty(ctx, partyID)
		if err != nil {
			return fmt.Errorf("failed to get party: %w", err)
		}
		if party == nil {
			return fmt.Errorf("party not found")
		}

		// Validate host permissions
		if !party.IsHost(hostID) {
			return fmt.Errorf("only the host can finalize nominations")
		}

		// Validate party phase
		if party.Phase != PhaseNominating {
			return fmt.Errorf("party is not in nominating phase")
		}

		// Check if there are any nominations
		if len(party.NominationPool) == 0 {
			return fmt.Errorf("no movies have been nominated")
		}

		// Move to ranking phase
		party.Phase = PhaseRanking
		party.CurrentNomination = nil // Clear any ongoing nomination

		// Initialize submissions map
		if party.Submissions == nil {
			party.Submissions = make(map[string][]string)
		}

		// Save updated party
		if err := s.redis.SaveParty(ctx, party); err != nil {
			return fmt.Errorf("failed to save party: %w", err)
		}

		updatedParty = party
		return nil
	})

	return updatedParty, err
}

// SubmitRanking handles final ranking submission and calculates results
func (s *Service) SubmitRanking(ctx context.Context, partyID, userID string, rankings []string) (*Party, error) {
	var updatedParty *Party

	err := s.WithLock(ctx, partyID, func(ctx context.Context) error {
		// Get current party state
		party, err := s.redis.GetParty(ctx, partyID)
		if err != nil {
			return fmt.Errorf("failed to get party: %w", err)
		}
		if party == nil {
			return fmt.Errorf("party not found")
		}

		// Validate party phase
		if party.Phase != PhaseRanking {
			return fmt.Errorf("ranking is not open")
		}

		// Validate rankings length matches nominated movies
		if len(rankings) != len(party.NominationPool) {
			return fmt.Errorf("ranking must include all %d nominated movies", len(party.NominationPool))
		}

		// Validate that all movie IDs in the ranking are valid
		nominatedMovieIDs := make(map[string]bool)
		for _, movie := range party.NominationPool {
			nominatedMovieIDs[movie.ID] = true
		}

		for _, movieID := range rankings {
			if !nominatedMovieIDs[movieID] {
				return fmt.Errorf("invalid movie ID in ranking: %s", movieID)
			}
		}

		// Store user's ranking
		if party.Submissions == nil {
			party.Submissions = make(map[string][]string)
		}
		party.Submissions[userID] = rankings

		// Check if all participants have submitted rankings
		allRankingsSubmitted := len(party.Submissions) == len(party.Participants)

		if allRankingsSubmitted {
			// Calculate final results using RCV
			winner, err := CalculateWinner(party.NominationPool, party.Submissions)
			if err != nil {
				log.Printf("Error calculating RCV winner for party %s: %v", partyID, err)
			} else {
				party.Winner = winner
			}
			party.Phase = PhaseFinished
		}

		// Save updated party
		if err := s.redis.SaveParty(ctx, party); err != nil {
			return fmt.Errorf("failed to save party: %w", err)
		}

		updatedParty = party
		return nil
	})

	return updatedParty, err
}
