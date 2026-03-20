# Internal Packages (`internal/`)

This directory contains the core backend logic for the r/place server, organized as Go packages following the standard Go project layout. The `internal/` convention means these packages are private to this module and cannot be imported by external projects.

## Directory Layout


```
internal/
├── canvas/
│   └── canvas.go       # Canvas data structure and pixel operations
└── server/
    ├── http.go         # HTTP handler for canvas snapshots
    └── websocket.go    # WebSocket handler for real-time pixel updates
```

## Packages

### `canvas` — Canvas State Management

**File:** `canvas/canvas.go`

Defines the `Canvas` struct, which holds the shared pixel grid. The canvas is a 2D array of hex color strings (e.g. `"#FF0000"`) with dimensions defaulting to 1000x1000 (the frontend renders a 100x100 view).

**Key type:**

```go
type Canvas struct {
    mu     sync.RWMutex    // Protects concurrent pixel access
    Width  int
    Height int
    Pixels [][]string      // Pixels[y][x] = hex color
}
```

**Functions and methods:**

- **`NewCanvas(width, height int) *Canvas`** — Creates a new canvas initialized to all white (`#FFFFFF`). Returns `nil` if dimensions are invalid.

- **`SetPixel(x, y int, color string) bool`** — Sets the color of a single pixel. Validates that coordinates are within bounds. Uses an exclusive write lock (`mu.Lock()`). Returns `true` on success, `false` if out of bounds.

- **`Snapshot() [][]string`** — Returns a deep copy of the entire pixel grid. Uses a read lock (`mu.RLock()`), allowing multiple concurrent snapshots without blocking each other.

**Concurrency model:** The `sync.RWMutex` allows multiple goroutines to read the canvas simultaneously (for snapshots) while ensuring exclusive access for writes (pixel placements). This means snapshot requests never block each other, and writes are serialized to prevent race conditions.

---

### `server` — HTTP and WebSocket Handlers

#### `server/http.go` — Snapshot Endpoint

Handles the `GET /snapshot` HTTP endpoint, which returns the full canvas state as JSON.

**Key type:**

```go
type HttpHandler struct {
    canvas *canvas.Canvas
}
```

**Response format:**

```json
{
  "width": 1000,
  "height": 1000,
  "pixels": [
    ["#FFFFFF", "#FF0000", ...],
    ...
  ]
}
```

The handler sets `Access-Control-Allow-Origin: *` to allow cross-origin requests from the React frontend during development.

**Constructor:** `NewHttpHandler(canvas *canvas.Canvas) *HttpHandler`

---

#### `server/websocket.go` — Real-Time Pixel Updates

Handles the `/ws` WebSocket endpoint. Each client connection is upgraded from HTTP and managed as a persistent bidirectional channel.

**Key types:**

```go
type WSHandler struct {
    canvas  *canvas.Canvas
    mu      sync.Mutex               // Protects the clients map
    clients map[*websocket.Conn]bool // Set of active connections
}

type PixelUpdate struct {
    X     int    `json:"x"`
    Y     int    `json:"y"`
    Color string `json:"color"`
}
```

**How it works:**

1. **Connection:** When a client connects, the HTTP request is upgraded to a WebSocket using `gorilla/websocket`. The connection is added to the `clients` map.

2. **Message loop:** The handler reads messages in a loop. Each message is expected to be a JSON `PixelUpdate` with `{x, y, color}`. The server calls `canvas.SetPixel()` to validate and apply the update.

3. **Broadcast:** After a successful pixel update, the raw message bytes are broadcast to every connected client via `broadcast()`. If sending to a client fails, that client is removed and its connection is closed.

4. **Disconnect:** When a client disconnects (read error), the connection is removed from the map and closed via a deferred cleanup.

**Constructor:** `NewWSHandler(canvas *canvas.Canvas) *WSHandler`

**Concurrency model:** The `clients` map is protected by its own `sync.Mutex`, separate from the canvas mutex. The broadcast function acquires this lock for the duration of the send loop. Each WebSocket connection runs in its own goroutine (spawned by Go's HTTP server).

## How It All Connects

The entry point (`cmd/main.go`) wires everything together:

```go
c := canvas.NewCanvas(canvas.DefaultCanvasWdith, canvas.DefaultCanvasHeight)
httpHandler := server.NewHttpHandler(c)
wsHandler := server.NewWSHandler(c)

http.HandleFunc("/snapshot", httpHandler.GetSnapshot)
http.HandleFunc("/ws", wsHandler.HandleWS)
http.ListenAndServe(":8080", nil)
```

Both handlers share a single `Canvas` instance. The HTTP handler reads from it (snapshots) while the WebSocket handler reads and writes (pixel placements and broadcasts). The `RWMutex` inside `Canvas` coordinates access between them.

## Planned Packages (Not Yet Implemented)

The following packages are planned for future demos as the system evolves into a distributed architecture:

| Package | Purpose |
|---------|---------|
| `election/` | Bully algorithm for leader election with heartbeat-based failure detection |
| `replication/` | Push-based replication of pixel updates from leader to follower replicas |
| `snapshot/` | Periodic canvas persistence to disk and crash recovery |
| `peer/` | gRPC client/server for replica-to-replica communication |
