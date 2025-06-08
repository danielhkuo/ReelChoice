
# ReelChoice

![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg) ![Build Status](https://img.shields.io/badge/build-passing-brightgreen) ![Svelte](https://img.shields.io/badge/Svelte-4-orange) ![Go](https://img.shields.io/badge/Go-1.21-blue)

ReelChoice is a real-time, collaborative movie selection tool designed to end the "what should we watch?" debate. It provides a democratic, two-phase process for groups to nominate and rank movie choices, ensuring everyone's preference is counted.

## Key Features

*   **Real-Time Collaboration:** See suggestions, votes, and results appear instantly without refreshing the page, powered by WebSockets.
*   **Two-Phase Selection:**
    1.  **Nomination Phase:** Anyone can suggest a movie. The group votes "Yay" or "Nay" to decide if it makes the final ballot.
    2.  **Ranking Phase:** Using the nominated movies, participants use Ranked-Choice Voting (RCV) to order their preferences.
*   **Ranked-Choice Voting (RCV):** A fair voting system that finds the most agreeable choice, even with diverse tastes.
*   **Host-Led Sessions:** A party "Host" controls the flow, deciding when to finalize nominations and start the final vote.
*   **Secure & Private:** Parties can be protected with an optional password.

## Tech Stack & Architecture

ReelChoice uses a modern, decoupled architecture for high performance and scalability.

*   **Frontend:** **SvelteKit** for a fast, compiled, and truly reactive user interface.
*   **Backend:** **Go** for a high-performance, concurrent API and WebSocket server.
*   **Real-Time State:** **Redis** for managing the ephemeral state of active parties (votes, participants) and for caching.
*   **Persistent Storage:** **PostgreSQL** for storing long-term data like party metadata and final results.

```
+--------------------+   HTTP/S (REST API)   +----------------------+
|                    |---------------------->|                      |
|  Client Browser    |                       |                      |
| (SvelteKit Frontend)|<----------------------|   Go Backend Service |
|                    |  WebSocket Messages   |                      |
|                    |                       |                      |
+--------------------+                       +----------+-----------+
                                                        |
                                +-----------------------+-----------------------+
                                |                                               |
              +-----------------v------------------+         +------------------v-----------------+
              |                                  |         |                                  |
              |  PostgreSQL Database             |         |  Redis                           |
              |  (Persistent Storage)            |         |  (Real-Time State & Cache)       |
              +----------------------------------+         +----------------------------------+
```

## Production Deployment

This application is designed to be deployed using containers.

### Prerequisites

*   [Docker](https://www.docker.com/) & [Docker Compose](https://docs.docker.com/compose/)
*   A registered API key from [TheMovieDB (TMDB)](https://www.themoviedb.org/documentation/api).

### Configuration

Before building, create a `.env` file in the project root by copying `.env.example`.

**`/backend/.env`**
```
# Port for the Go server to listen on
PORT=8080
# Full connection string for your PostgreSQL database
DATABASE_URL="postgres://user:password@host:port/dbname"
# Full connection string for your Redis instance
REDIS_URL="redis://user:password@host:port/0"
# Your secret API key from TMDB
TMDB_API_KEY="your_tmdb_api_key_here"
```

### Running with Docker Compose

The included `docker-compose.yml` file will build the production images for the frontend and backend, and run them alongside Postgres and Redis services.

1.  **Configure Environment:** Ensure your `.env` files in `/backend` are correctly configured.
2.  **Build and Run:**
    ```bash
    docker-compose up --build
    ```
3.  The frontend will be accessible at `http://localhost:80` (or the port you map).

## Development Setup

For local development without containers:

### Prerequisites
*   Go (v1.20+)
*   Node.js (v18+)
*   Running instances of PostgreSQL and Redis.

### 1. Backend
```bash
cd backend
go mod tidy
# Create a .env file with your local dev credentials
go run main.go
```

### 2. Frontend
```bash
cd frontend
npm install
npm run dev
```
The frontend will be available at `http://localhost:5173`. API requests to `/api` are automatically proxied to the backend at `http://localhost:8080` via the `vite.config.js` settings.

## Contributing

We welcome contributions! Please feel free to submit a Pull Request or open an issue for bugs, feature requests, or questions.

1.  Fork the repository.
2.  Create your feature branch (`git checkout -b feature/AmazingFeature`).
3.  Commit your changes (`git commit -m 'Add some AmazingFeature'`).
4.  Push to the branch (`git push origin feature/AmazingFeature`).
5.  Open a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.