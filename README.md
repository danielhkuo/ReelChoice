# ReelChoice

Welcome to ReelChoice! This project is a collaborative movie selection tool built on a modern, high-performance stack:

*   **Frontend:** [Astro](https://astro.build/) with [React](https://react.dev/) for interactive UI components.
*   **Backend:** [Go](https://go.dev/) for a fast, concurrent, and reliable API.

This combination provides an incredibly fast-loading user experience thanks to Astro's static-first approach, with dynamic, real-time features powered by React "islands" and a robust Go backend.

## Project Structure

This project is a monorepo, with the frontend and backend code living in separate directories but managed in the same repository. This simplifies development, versioning, and deployment.

```
/reelchoice/
├── .gitignore          # A single gitignore for the whole project
├── README.md           # Project-level instructions
│
├── /backend/           # All Go code
│   ├── main.go         # Backend server entry point
│   ├── go.mod          # Go module definition
│   ├── go.sum          # Go module checksums
│   ├── /api/           # API route handlers
│   └── /websocket/     # WebSocket connection logic
│
└── /frontend/          # All Astro + React code
    ├── package.json
    ├── astro.config.mjs  # Astro configuration
    ├── /public/          # Static assets (images, fonts)
    └── /src/
        ├── /components/  # Interactive React components (.jsx)
        ├── /layouts/     # Astro page layouts (.astro)
        └── /pages/       # Astro pages (.astro)
```

## System Architecture

The ReelChoice application is built on a decoupled, real-time architecture designed for performance and scalability. The frontend client communicates with the backend Go service, which acts as the central hub for all logic, state management, and data persistence.

```
+--------------------+   HTTP/S (REST API)   +----------------------+
|                    |---------------------->|                      |
|  Client Browser    |                       |                      |
| (Astro with React  |<----------------------|   Go Backend Service |
|      Islands)      |  WebSocket Messages   |                      |
|                    |                       |                      |
+--------------------+                       +----------+-----------+
                                                        |
                                +-----------------------+-----------------------+
                                |                                               |
              +-----------------v------------------+         +------------------v-----------------+
              |                                  |         |                                  |
              |  PostgreSQL Database             |         |  Redis                           |
              |  (Persistent Storage)            |         |  (Real-Time State & Cache)       |
              |                                  |         |                                  |
              |  - Party Metadata (name, etc.)   |         |  - Active Party State (JSON)     |
              |  - Final Winning Movie           |         |  - Live Vote Counts              |
              |                                  |         |  - TMDB API Response Cache       |
              +----------------------------------+         +----------------------------------+
```

### Component Breakdown

*   **Frontend (Astro + React):** The user-facing application.
    *   **Astro** is used to build the static "shell" of the pages (layouts, non-interactive content). This results in extremely fast initial page loads.
    *   **React** "islands" are used for all interactive components (the nomination queue, voting buttons, ranking list). These components hydrate on the client-side to manage dynamic UI and communicate with the backend.

*   **Backend (Go):** The single source of truth for the application. It serves two primary roles:
    1.  **REST API Server:** Exposes standard HTTP endpoints for non-real-time actions like creating a new party or searching for a movie.
    2.  **WebSocket Hub:** Manages persistent WebSocket connections for each active party. When a user takes an action (like voting), the backend updates the state and broadcasts the new state to all clients in that party.

*   **PostgreSQL Database:** This is our system for **data at rest**. It stores long-term, persistent information that needs to survive server restarts.
    *   **Examples:** Party details (name, creation date), final winning movie results. It does *not* store every single nomination vote, as that is ephemeral.

*   **Redis:** This is our high-speed system for **data in motion**. It is used for managing the ephemeral, real-time state of active parties and for caching.
    *   **Live State:** The entire state of an active party (participants, nominations, current vote, phase) is stored as a single object in Redis. This allows for incredibly fast reads and writes.
    *   **Caching:** Backend search requests to the external TMDB API are cached in Redis to improve speed and avoid hitting rate limits.
    *   **Pub/Sub (for scaling):** Redis's Pub/Sub capabilities can be used to broadcast messages between multiple backend server instances if the application needs to scale horizontally.

*   **TheMovieDB (TMDB) API:** This is the external service used to fetch movie data. The Go backend acts as a **proxy** for all TMDB requests. The client **never** communicates directly with TMDB; this protects the API key and allows our backend to implement efficient caching.

## Getting Started

Follow these steps to get the development environment running on your local machine.

### Prerequisites

Before you begin, ensure you have the following installed:
*   [Go](https://go.dev/doc/install) (version 1.20 or newer)
*   [Node.js](https://nodejs.org/en) (version 18 or newer)

### 1. Backend Setup (Go)

First, set up and run the Go API server.

1.  **Navigate to the backend directory:**
    ```bash
    cd backend
    ```

2.  **Install Dependencies:** This command syncs the dependencies listed in the `go.mod` file.
    ```bash
    go mod tidy
    ```

3.  **Run the Server:**
    ```bash
    go run main.go
    ```
    The backend server will start and listen on `http://localhost:8080`.

    > **Note for Go developers:** For a better development experience with hot-reloading, consider using a tool like [Air](https://github.com/cosmtrek/air).

### 2. Frontend Setup (Astro + React)

Next, set up the Astro frontend in a separate terminal.

1.  **Navigate to the frontend directory:**
    *(From the project root)*
    ```bash
    cd frontend
    ```

2.  **Install Dependencies:**
    ```bash
    npm install
    ```

3.  **Run the Development Server:**
    ```bash
    npm run dev
    ```
    The frontend development server will be available at `http://localhost:4321`. It includes Hot Module Replacement (HMR) for a fast feedback loop.

### 3. Connecting Frontend & Backend (Proxy)

During development, the Astro frontend (`:4321`) and Go backend (`:8080`) run on different ports. To avoid browser CORS (Cross-Origin Resource Sharing) errors, the Astro dev server is configured to proxy API requests.

The `frontend/astro.config.mjs` file contains a proxy rule:

```javascript
// frontend/astro.config.mjs
import { defineConfig } from 'astro/config';
import react from '@astrojs/react';

export default defineConfig({
  integrations: [react()],
  server: {
    proxy: {
      // Proxy all /api requests to the Go backend
      '/api': 'http://localhost:8080',
    }
  }
});
```

This configuration means you don't need to change anything. It's already set up for you.

## How to Run

With both servers running, you can now start development:

1.  **Open your browser** to `http://localhost:4321`.
2.  Any `fetch` requests made from your React components to an endpoint like `/api/party` will be correctly forwarded to your Go backend.
3.  You're all set! Start building your components in `/frontend/src/components` and your API logic in `/backend/api`.
