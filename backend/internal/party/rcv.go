package party

import (
	"fmt"
	"log"
)

// CalculateWinner implements Ranked-Choice Voting (RCV) to determine the winning movie
// RCV works by eliminating movies with the fewest first-choice votes iteratively
// until one movie has a majority (>50%) of the remaining votes
func CalculateWinner(pool []Movie, submissions map[string][]string) (*Movie, error) {
	if len(pool) == 0 {
		return nil, fmt.Errorf("no movies in nomination pool")
	}

	if len(submissions) == 0 {
		return nil, fmt.Errorf("no voting submissions received")
	}

	// Handle edge case: only one movie
	if len(pool) == 1 {
		return &pool[0], nil
	}

	// Create a map of movie ID to Movie for quick lookup
	movieMap := make(map[string]Movie)
	for _, movie := range pool {
		movieMap[movie.ID] = movie
	}

	// Validate all submissions contain only valid movie IDs
	for userID, ranking := range submissions {
		for _, movieID := range ranking {
			if _, exists := movieMap[movieID]; !exists {
				log.Printf("Warning: User %s voted for invalid movie ID %s", userID, movieID)
			}
		}
	}

	// Track which movies are still in the running
	activeMovies := make(map[string]bool)
	for _, movie := range pool {
		activeMovies[movie.ID] = true
	}

	totalVoters := len(submissions)
	majorityThreshold := totalVoters/2 + 1

	log.Printf("Starting RCV calculation with %d movies and %d voters (majority threshold: %d)",
		len(pool), totalVoters, majorityThreshold)

	// Run RCV rounds until we have a winner
	round := 1
	for len(activeMovies) > 1 {
		log.Printf("RCV Round %d: %d movies remaining", round, len(activeMovies))

		// Count first-choice votes for each active movie
		voteCounts := make(map[string]int)
		for _, movie := range pool {
			if activeMovies[movie.ID] {
				voteCounts[movie.ID] = 0
			}
		}

		// For each submission, find the highest-ranked active movie
		for userID, ranking := range submissions {
			firstChoice := getFirstActiveChoice(ranking, activeMovies)
			if firstChoice != "" {
				voteCounts[firstChoice]++
			} else {
				log.Printf("Warning: User %s has no valid choices remaining", userID)
			}
		}

		// Check if any movie has a majority
		for movieID, votes := range voteCounts {
			if votes >= majorityThreshold {
				winner := movieMap[movieID]
				log.Printf("RCV Winner: %s with %d votes (%.1f%%)",
					winner.Title, votes, float64(votes)/float64(totalVoters)*100)
				return &winner, nil
			}
		}

		// No majority found, eliminate the movie with the fewest votes
		minVotes := totalVoters + 1
		var movieToEliminate string

		for movieID, votes := range voteCounts {
			log.Printf("  %s: %d votes", movieMap[movieID].Title, votes)
			if votes < minVotes {
				minVotes = votes
				movieToEliminate = movieID
			}
		}

		if movieToEliminate == "" {
			// This shouldn't happen, but safeguard against infinite loop
			break
		}

		// Eliminate the movie with fewest votes
		eliminatedMovie := movieMap[movieToEliminate]
		delete(activeMovies, movieToEliminate)
		log.Printf("  Eliminated: %s with %d votes", eliminatedMovie.Title, minVotes)

		round++
	}

	// If we're down to one movie, it wins
	if len(activeMovies) == 1 {
		for movieID := range activeMovies {
			winner := movieMap[movieID]
			log.Printf("RCV Winner by elimination: %s", winner.Title)
			return &winner, nil
		}
	}

	// This shouldn't happen, but return first movie as fallback
	log.Printf("Warning: RCV algorithm inconclusive, returning first movie")
	return &pool[0], nil
}

// getFirstActiveChoice returns the ID of the highest-ranked movie that's still active
func getFirstActiveChoice(ranking []string, activeMovies map[string]bool) string {
	for _, movieID := range ranking {
		if activeMovies[movieID] {
			return movieID
		}
	}
	return ""
}
