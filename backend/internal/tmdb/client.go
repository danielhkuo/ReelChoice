package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/reelchoice/backend/internal/database"
	"github.com/reelchoice/backend/internal/party"
)

// Client represents a TMDB API client
type Client struct {
	apiKey      string
	redisClient *database.RedisClient
	httpClient  *http.Client
	baseURL     string
}

// TMDBMovieResult represents a movie result from TMDB API
type TMDBMovieResult struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	ReleaseDate string  `json:"release_date"`
	PosterPath  string  `json:"poster_path"`
	Overview    string  `json:"overview"`
	VoteAverage float64 `json:"vote_average"`
}

// TMDBSearchResponse represents the search response from TMDB
type TMDBSearchResponse struct {
	Page    int               `json:"page"`
	Results []TMDBMovieResult `json:"results"`
	Total   int               `json:"total_results"`
}

// NewClient creates a new TMDB client
func NewClient(apiKey string, redisClient *database.RedisClient) *Client {
	return &Client{
		apiKey:      apiKey,
		redisClient: redisClient,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.themoviedb.org/3",
	}
}

// SearchMovies searches for movies using the TMDB API
func (c *Client) SearchMovies(ctx context.Context, query string) ([]party.Movie, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	// Check cache first
	cachedData, err := c.redisClient.GetCachedTMDBData(ctx, query)
	if err == nil && cachedData != nil {
		var movies []party.Movie
		if err := json.Unmarshal(cachedData, &movies); err == nil {
			return movies, nil
		}
	}

	// Build search URL
	searchURL := fmt.Sprintf("%s/search/movie", c.baseURL)
	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("query", query)
	params.Set("page", "1")
	params.Set("include_adult", "false")

	fullURL := fmt.Sprintf("%s?%s", searchURL, params.Encode())

	// Make HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API returned status %d", resp.StatusCode)
	}

	// Parse response
	var searchResponse TMDBSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our movie format
	movies := make([]party.Movie, 0, len(searchResponse.Results))
	for _, result := range searchResponse.Results {
		// Extract year from release date
		year := ""
		if result.ReleaseDate != "" && len(result.ReleaseDate) >= 4 {
			year = result.ReleaseDate[:4]
		}

		// Build poster URL
		posterPath := ""
		if result.PosterPath != "" {
			posterPath = fmt.Sprintf("https://image.tmdb.org/t/p/w500%s", result.PosterPath)
		}

		movie := party.Movie{
			ID:         strconv.Itoa(result.ID),
			Title:      result.Title,
			Year:       year,
			PosterPath: posterPath,
		}
		movies = append(movies, movie)
	}

	// Cache the result
	if len(movies) > 0 {
		movieData, err := json.Marshal(movies)
		if err == nil {
			c.redisClient.SetCachedTMDBData(ctx, query, movieData)
		}
	}

	return movies, nil
}

// GetMovieDetails gets detailed information about a specific movie
func (c *Client) GetMovieDetails(ctx context.Context, tmdbID string) (*party.Movie, error) {
	if tmdbID == "" {
		return nil, fmt.Errorf("movie ID cannot be empty")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("movie:%s", tmdbID)
	cachedData, err := c.redisClient.GetCachedTMDBData(ctx, cacheKey)
	if err == nil && cachedData != nil {
		var movie party.Movie
		if err := json.Unmarshal(cachedData, &movie); err == nil {
			return &movie, nil
		}
	}

	// Build movie details URL
	movieURL := fmt.Sprintf("%s/movie/%s", c.baseURL, tmdbID)
	params := url.Values{}
	params.Set("api_key", c.apiKey)

	fullURL := fmt.Sprintf("%s?%s", movieURL, params.Encode())

	// Make HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API returned status %d", resp.StatusCode)
	}

	// Parse response
	var movieResult TMDBMovieResult
	if err := json.NewDecoder(resp.Body).Decode(&movieResult); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract year from release date
	year := ""
	if movieResult.ReleaseDate != "" && len(movieResult.ReleaseDate) >= 4 {
		year = movieResult.ReleaseDate[:4]
	}

	// Build poster URL
	posterPath := ""
	if movieResult.PosterPath != "" {
		posterPath = fmt.Sprintf("https://image.tmdb.org/t/p/w500%s", movieResult.PosterPath)
	}

	movie := &party.Movie{
		ID:         tmdbID,
		Title:      movieResult.Title,
		Year:       year,
		PosterPath: posterPath,
	}

	// Cache the result
	movieData, err := json.Marshal(movie)
	if err == nil {
		c.redisClient.SetCachedTMDBData(ctx, cacheKey, movieData)
	}

	return movie, nil
}
