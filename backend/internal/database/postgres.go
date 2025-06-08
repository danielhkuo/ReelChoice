package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresClient wraps the PostgreSQL connection pool
type PostgresClient struct {
	pool *pgxpool.Pool
}

// NewPostgresClient creates a new PostgreSQL client connection
func NewPostgresClient(databaseURL string) (*PostgresClient, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	pool, err := pgxpool.New(context.Background(), config.ConnString())
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresClient{pool: pool}, nil
}

// Close closes the PostgreSQL connection pool
func (p *PostgresClient) Close() {
	p.pool.Close()
}

// Ping tests the database connection
func (p *PostgresClient) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// TODO: Add methods for persistent storage of party results, user data, etc.
// This will be implemented when we need to store final results and historical data
