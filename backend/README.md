# ReelChoice Backend

A high-performance, scalable Go backend service for the ReelChoice collaborative movie selection application. This service is architected for production workloads, featuring a stateless design, distributed state management, and robust concurrency control.

## Architecture

The backend follows a clean, modular architecture with a clear separation of concerns, enabling maintainability and horizontal scaling.

```
backend/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point & server setup
├── internal/
│   ├── api/                     # HTTP REST API handlers (transport layer)
│   │   └── handlers.go
│   ├── config/                  # Configuration management
│   │   └── config.go
│   ├── database/                # Database clients (Redis & PostgreSQL)
│   │   ├── redis.go
│   │   └── postgres.go
│   ├── party/                   # Core business logic and domain
│   │   ├── auth.go              # Scalable, Redis-backed authentication
│   │   ├── protocol.go          # WebSocket message definitions
│   │   ├── rcv.go               # Ranked-Choice Voting algorithm
│   │   ├── service.go           # Business logic service layer
│   │   └── state.go             # Core data structures (domain models)
│   ├── tmdb/                    # TheMovieDB API client
│   │   └── client.go
│   └── websocket/               # Real-time communication hub (transport layer)
│       └── hub.go
├── go.mod                       # Go module definition
├── go.sum                       # Dependency checksums
├── example.env                  # Environment variables template
└── README.md                    # This file
```

## Features

### ✅ Completed & Production-Ready

#### Phase 1: Party Lifecycle & Real-Time Connectivity
- **REST API Endpoints:**
  - `POST /api/party`: Create a new party.
  - `GET /api/party/{id}`: Get party information.
  - `POST /api/party/{id}/join`: Join a party with a username.
  - `POST /api/party/{id}/start-nomination`: Start nomination phase (host only).
  - `GET /api/movies/search`: Search movies via the TMDB API.
  - `GET /api/health`: Health check endpoint.
- **Stateless & Scalable Authentication:**
  - Secure tokens are generated upon party creation/join and stored in Redis.
  - The authentication layer is stateless, allowing for horizontal scaling of the backend service.
- **WebSocket Real-Time Communication:**
  - A dedicated hub manages WebSocket connections, delegating all business logic to the Party Service.
  - Real-time party state updates are broadcast efficiently to all members.
  - A Ping/Pong heartbeat system ensures connection health and cleans up stale connections.

#### Phase 2: Movie Nomination System
- **TMDB Integration:**
  - Movie search and details retrieval are cached in Redis to reduce latency and avoid API rate limits.
- **Nomination Workflow:**
  - Users can suggest movies, triggering a real-time "Yay/Nay" vote for all participants.
  - **Concurrency Safe:** All state-mutating operations are protected by a Redis-based distributed lock, preventing race conditions.
  - Nominations are approved by majority vote and added to the final ballot.
  - The host has exclusive control over finalizing the nomination phase.

#### Phase 3: Ranked-Choice Voting (RCV)
- **RCV Algorithm:**
  - A pure-function implementation of instant-runoff voting determines the winner by iteratively eliminating movies with the fewest votes.
- **Ranking System:**
  - Participants submit their ranked preferences via WebSocket.
  - The backend validates all submissions against the nominated movie pool.
  - The winner is automatically calculated and broadcast once all participants have voted.

## Technology Stack

- **Language:** Go 1.24+
- **Web Framework:** Chi v5 (lightweight, fast HTTP router)
- **WebSockets:** Gorilla WebSocket
- **State & Auth Storage:** Redis (ephemeral party state, caching, auth tokens)
- **Distributed Locking:** Redis (`SETNX`)
- **Persistent Storage:** PostgreSQL (for future use: historical data, user accounts)
- **External API:** TheMovieDB (TMDB) for movie data
- **Configuration:** Environment variables with godotenv

## Quick Start

### Prerequisites

- Go 1.24 or later
- A running Redis server
- A running PostgreSQL server
- A TMDB API key ([Get one here](https://www.themoviedb.org/documentation/api))

### Installation

1.  **Clone and set up:**
    ```bash
    cd backend
    go mod tidy
    ```

2.  **Configure environment:**
    ```bash
    cp example.env .env
    # Edit .env with your actual database, Redis, and TMDB values
    ```

3.  **Run the server:**
    ```bash
    go run cmd/server/main.go
    ```

    Or build and run a production binary:
    ```bash
    go build -o reelchoice-backend ./cmd/server
    ./reelchoice-backend
    ```

## API Documentation

### Authentication & Security

ReelChoice uses a stateless, token-based authentication system for secure and scalable operations.

1.  **Creating/Joining:** When you create or join a party, the API returns an `auth_token`.
2.  **Token Storage:** This token is securely stored in Redis with a 24-hour TTL, making the auth system stateless.
3.  **WebSocket Connection:** You must use this token to authenticate your WebSocket connection: `?token={yourAuthToken}`.
4.  **Host Actions:** Host-only REST endpoints require the token in the `Authorization: Bearer {yourAuthToken}` header.
5.  **Validation:** All incoming tokens are validated against Redis to ensure the session is active and authorized for the requested party.

### REST Endpoints

| Method | Endpoint                           | Description                   | Auth Required |
| :----- | :--------------------------------- | :---------------------------- | :------------ |
| `POST` | `/api/party`                       | Create a new party            | No            |
| `GET`  | `/api/party/{id}`                  | Get party information         | No            |
| `POST` | `/api/party/{id}/join`             | Join a party                  | No            |
| `POST` | `/api/party/{id}/start-nomination` | Start the nomination phase    | Yes (Host)    |
| `GET`  | `/api/movies/search?q={query}`     | Search movies via TMDB        | No            |
| `GET`  | `/api/health`                      | Health check for the service  | No            |

### WebSocket Protocol

**Connection:** `ws://localhost:8080/ws/party/{partyID}?token={yourAuthToken}`

#### Message Types

| Type                     | Direction         | Payload                                | Description                                |
| :----------------------- | :---------------- | :------------------------------------- | :----------------------------------------- |
| `ping`                   | Client → Server   | `{}`                                   | Heartbeat request to keep connection alive |
| `pong`                   | Server → Client   | `{"timestamp": number}`                | Heartbeat response from the server         |
| `search_movies`          | Client → Server   | `{"query": "string"}`                  | Search for movies via TMDB                 |
| `search_results`         | Server → Client   | `{"query": "string", "movies": [...]}` | Movie search results                       |
| `suggest_movie`          | Client → Server   | `{"tmdb_id": "string"}`                | Suggest a movie for nomination             |
| `vote_nomination`        | Client → Server   | `{"vote": "yay"\|"nay"}`                | Vote on the current nomination             |
| `finalize_nominations`   | Client → Server   | `{}`                                   | End nomination phase (host only)           |
| `submit_ranking`         | Client → Server   | `{"ranks": ["id1", "id2"]}`            | Submit ranked preferences                  |
| `party_update`           | Server → Client   | `{"party": {...}}`                     | Broadcasts the entire updated party state  |
| `error`                  | Server → Client   | `{"error": "string"}`                  | Informs the client of an error             |

## Development

### Architectural Overview

-   **`cmd/server/`**: The application entry point. Initializes dependencies (DB, Redis, services) and wires up the HTTP router.
-   **`internal/party/service.go`**: The core of the application. **This service layer contains all business logic**, ensuring that the API and WebSocket handlers remain thin transport layers.
-   **`internal/websocket/`**: The `Hub` manages the connection lifecycle and message transport. It receives raw WebSocket messages and delegates them to the `Party Service` for processing.
-   **`internal/api/`**: Contains the REST API handlers. These are thin wrappers that parse HTTP requests and call the appropriate methods on the `Party Service`.
-   **`internal/party/auth.go`**: Implements a scalable authentication system by storing and validating tokens in Redis, decoupling session state from the application server.

### Key Design Decisions

1.  **Stateless Services:** The application logic is stateless. All state (parties, auth tokens) is externalized to Redis, allowing for easy horizontal scaling.
2.  **Service Layer Decoupling:** Business logic is strictly contained within the `party.Service`, separating it from the HTTP and WebSocket transport layers.
3.  **Distributed Locking for Concurrency:** All read-modify-write operations on party state are protected by a Redis-based distributed lock (`SETNX`) to prevent race conditions and ensure data consistency.
4.  **Redis-Backed Authentication:** User session tokens are stored in Redis with a TTL, providing a scalable and robust authentication mechanism.
5.  **Efficient Caching:** TMDB API responses are cached in Redis to minimize external calls, reduce latency, and avoid rate-limiting issues.

### Testing

```bash
# Run all tests
go test ./...

# Run the server with the race detector enabled to check for concurrency issues
go run -race cmd/server/main.go
```

## Production Deployment

The application is built for containerized, horizontally-scaled environments.

1.  **Docker Build:** A `Dockerfile` can be used to build a lightweight production image.
    ```bash
    docker build -t reelchoice-backend .
    ```
2.  **Environment Variables:** All configuration is managed via environment variables, adhering to 12-Factor App principles.
3.  **Health Checks:** Use the `GET /api/health` endpoint for load balancer and container orchestrator health checks.
4.  **Scaling:** The stateless nature of the service allows you to run multiple instances behind a load balancer without issue.

## Contributing

1.  Follow Go best practices and conventions.
2.  Add tests for any new functionality.
3.  Update documentation for API changes.
4.  Use meaningful commit messages.

## License

This project is licensed under the MIT License - see the root project LICENSE file for details.