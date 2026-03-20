# CPSC 559 Group Project - Distributed r/place Clone

**Team Members:** Ahsan Tariq, Jarin Thundathil, Marvellous Chukwukelu, Navpreet Singh, Nour Ajami

## Description

A real-time collaborative pixel canvas inspired by Reddit's r/place, built as a distributed systems course project for CPSC 559. Multiple clients can simultaneously connect and place colored pixels on a shared 100x100 canvas, with all changes broadcast in real-time to every connected user via WebSockets.

The backend is written in Go and the frontend is a single-page React application. The current implementation is a single-server proof of concept. Future milestones will introduce a distributed architecture with multiple Go replicas, leader election using the Bully algorithm, primary-backup replication, periodic snapshots for crash recovery, and client-side load balancing.

## Current Status

**Demo 2 - Single-Server Proof of Concept**

The system currently runs as a single Go server with a React frontend. This serves as the foundation for the full distributed system to be built in later demos.

### What Works
- Real-time collaborative pixel placement across multiple clients
- WebSocket-based broadcasting of all pixel updates
- HTTP snapshot endpoint for initial canvas state retrieval
- Two drawing modes: single pixel and adjustable brush (1x1 to 10x10)
- 12-color palette with live coordinate display (HUD)
- Thread-safe concurrent client handling using Go's `sync.RWMutex`

### Known Limitations
- **No persistence:** Canvas resets on server restart (in-memory only)
- **Single point of failure:** No replication or fault tolerance
- **No rate limiting:** Cooldown enforcement planned for later demos
- **Hardcoded addresses:** Frontend connects to `localhost:8080`

## Architecture Overview

```
┌─────────────────────┐         ┌──────────────────────────────┐
│   React Frontend    │         │       Go Backend             │
│   (localhost:3000)  │         │       (localhost:8080)       │
│                     │         │                              │
│  ┌───────────────┐  │  HTTP   │  ┌────────────────────────┐  │
│  │  App.js       │──│─────────│──│  GET /snapshot         │  │
│  │  (state hub)  │  │         │  │  (HttpHandler)         │  │
│  └───────┬───────┘  │         │  └────────────────────────┘  │
│          │          │         │                              │
│  ┌───────┴───────┐  │  WS     │  ┌────────────────────────┐  │
│  │ canvasService │──│─────────│──│  /ws                   │  │
│  │ (WebSocket)   │  │         │  │  (WSHandler)           │  │
│  └───────────────┘  │         │  └───────────┬────────────┘  │
│                     │         │              │               │
│  ┌───────────────┐  │         │  ┌───────────┴────────────┐  │
│  │ Canvas.js     │  │         │  │  Canvas (100x100)      │  │
│  │ Toolbar.js    │  │         │  │  sync.RWMutex          │  │
│  │ HUD.js        │  │         │  │  In-memory pixel grid  │  │
│  └───────────────┘  │         │  └────────────────────────┘  │
└─────────────────────┘         └──────────────────────────────┘
```

### Data Flow

1. **Client connects:** Fetches full canvas state via `GET /snapshot`
2. **WebSocket established:** Client opens persistent connection to `/ws`
3. **Pixel placed:** Client sends `{x, y, color}` over WebSocket
4. **Server validates:** Checks coordinates are within bounds (0-99)
5. **Canvas updated:** Server writes pixel to in-memory 2D array (mutex-protected)
6. **Broadcast:** Server sends the update to all connected WebSocket clients
7. **Clients render:** Each client applies the update to its local canvas view

### API Endpoints

| Endpoint | Protocol | Description |
|----------|----------|-------------|
| `GET /snapshot` | HTTP | Returns the full canvas state as JSON (`{width, height, pixels}`) |
| `/ws` | WebSocket | Bidirectional channel for pixel updates. Accepts `{x, y, color}` messages and broadcasts them to all clients |

### WebSocket Message Format

```json
{
  "x": 45,
  "y": 78,
  "color": "#FF0000"
}
```

## Prerequisites

- **Go** 1.22.4 or higher
- **Node.js** with npm

## Running the Project

### 1. Start the Backend

From the project root:

```bash
go run ./cmd/main.go
```

The server starts on **port 8080**.

### 2. Start the Frontend

In a separate terminal:

```bash
cd web
npm install
npm start
```

The React development server starts on **port 3000** and opens in your browser automatically.

### 3. Use It

Open `http://localhost:3000` in one or more browser tabs/windows. Each tab acts as an independent client. Select a color, choose pixel or brush mode, and click/drag on the canvas. All connected clients see updates in real-time.

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.22.4, `gorilla/websocket` |
| Frontend | React 19, HTML5 Canvas API |
| Transport | HTTP/1.1 (snapshots), WebSocket RFC 6455 (real-time) |
| Concurrency | Go goroutines, `sync.RWMutex` |

## Future Milestones

| Demo | Focus | Key Features |
|------|-------|--------------|
| Demo 3 | Distributed Replication | Multiple Go replicas, leader-follower architecture, Bully election algorithm, heartbeat failure detection |
| Demo 4 | Fault Tolerance | Periodic disk snapshots, crash recovery, leader re-election, client reconnection |
| Demo 5 | Production Polish | Client-side least-connections load balancing, cooldown enforcement (50 px / 10s per user) |

## Development Workflow

- **Branch naming:** `yourname/feature-name`
- **Integration branch:** `integration` (primary merge target)
- **Pull requests:** Submit to `integration`, not `main`

## License

MIT License. See [LICENSE](LICENSE) for details.

## Project Structure

We are going to be using the Standard Project Layout (This is a Golang standard layout), its considered the de facto convention for most go projects

```
559-project/
├── cmd/
│   └── main.go                  # Entry point - wires components, starts the server
│
├── internal/
│   ├── canvas/
│   │   └── canvas.go            # Canvas state (2D array, get/set pixel, snapshot)
│   │
│   ├── election/
│   │   └── bully.go             # Modified Bully algorithm, heartbeats (Not implemented yet)
│   │
│   ├── replication/
│   │   └── replication.go       # Broadcast pixel updates to followers (Not implemented yet)
│   │
│   ├── server/
│   │   ├── http.go              # HTTP handlers (GET /snapshot)
│   │   └── websocket.go         # WebSocket server (client connections)
│   │
│   ├── peer/
│   │   └── peer.go              # gRPC client/server (replica-to-replica)
│   │
│   └── snapshot/
│       └── snapshot.go          # Periodic snapshot to disk, recovery in case of failure (not implemented yet)
│
├── proto/
│   └── rplace.proto             # gRPC service definitions 
│
│
├── go.mod
├── go.sum
└── README.md
```
