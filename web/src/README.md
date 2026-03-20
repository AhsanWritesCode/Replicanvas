# Frontend Source (`web/src/`)

This directory contains the React frontend for the distributed r/place canvas. The app renders a 100x100 interactive pixel grid, connects to the Go backend over HTTP and WebSocket, and provides drawing tools for collaborative pixel placement.

## Directory Layout

```
src/
├── App.js                  # Root component and application state
├── App.css                 # Root layout styling
├── index.js                # React DOM entry point
├── index.css               # Global styles
├── components/
│   ├── Canvas.js           # Pixel canvas rendering and interaction
│   ├── Canvas.css
│   ├── Toolbar.js          # Drawing mode, brush size, and color controls
│   ├── Toolbar.css
│   ├── HUD.js              # Coordinate overlay display
│   └── HUD.css
└── services/
    └── canvasService.js    # Backend communication (HTTP + WebSocket)
```

## Component Overview

### App.js

The root component and central state hub. Manages:

- **Selected color** (`#FF0000` default) - the hex color used for pixel placement
- **Drawing mode** (`'pixel'` or `'brush'`) - single pixel vs. multi-pixel brush
- **Brush size** (1-10) - side length of the square brush
- **Cursor coordinates** - current mouse position on canvas, displayed in the HUD
- **Connection state** - tracks whether the WebSocket is connected

On mount, App fetches the initial canvas snapshot via HTTP (`GET /snapshot`) and then establishes a WebSocket connection (`/ws`). It passes state and callbacks down to all child components.

### Canvas.js

Renders the 100x100 pixel grid onto an HTML5 `<canvas>` element at 800x800 screen pixels (each logical pixel is 8x8 screen pixels). Handles all user interaction:

- **Click** places a pixel (or brush area) at the cursor position
- **Click + drag** paints continuously as the mouse moves
- **Mouse move** updates the HUD coordinates

The component maintains its own `canvasState` (a 100x100 2D string array of hex colors) initialized from the server snapshot. It subscribes to remote pixel updates via `canvasService.onPixelUpdate()` and applies them to the local state, so changes from other clients appear in real-time.

**Brush mode** places a square of pixels centered on the cursor. For a brush size of `n`, the square spans `n x n` pixels. Each individual pixel in the square is sent as a separate WebSocket message.

A light gray grid overlay is drawn on top of the pixels to visually separate them.

### Toolbar.js

Provides the drawing controls:

- **Mode selector** - toggle between "Single Pixel" and "Brush" modes. The active mode button is highlighted.
- **Brush size slider** - a range input from 1 to 10, only visible when brush mode is active. Displays the current size as `NxN`.
- **Color palette** - 12 predefined colors arranged in a grid:
  - Red, Orange, Yellow, Green, Cyan, Blue, Purple, Magenta, White, Black, Gray, Pink
  - The selected color is indicated with a golden border
- **Current color display** - shows the active color swatch and its hex code

### HUD.js

A small fixed overlay in the top-right corner of the screen that displays the current cursor coordinates on the canvas. Shows `(x, y)` when hovering over the canvas and `(--, --)` when the cursor is outside the canvas area. Uses a monospace font for readability.

## Services

### canvasService.js

A singleton class that handles all communication with the Go backend. It exposes:

- **`getSnapshot()`** - Fetches the full canvas state via `GET http://localhost:8080/snapshot`. Returns a JSON object with `{width, height, pixels}` where `pixels` is a 2D array of hex color strings. Called once on app initialization.

- **`connectWebSocket()`** - Opens a WebSocket connection to `ws://localhost:8080/ws`. Returns a Promise that resolves when the connection is established. Incoming messages are parsed as JSON and dispatched to all registered handlers.

- **`onPixelUpdate(handler)`** - Registers a callback to receive pixel updates from the server. Each update is `{x, y, color}`. Returns an unsubscribe function for cleanup.

- **`sendPixelUpdate(x, y, color)`** - Sends a pixel placement message over the WebSocket. Only sends if the connection is in the OPEN state.

- **`disconnect()`** - Closes the WebSocket connection. Called on App unmount.

**Backend URLs are hardcoded** to `localhost:8080`. Future demos will add support for multiple server endpoints and client-side load balancing.

## Data Flow

```
App mounts
  │
  ├─→ canvasService.getSnapshot()       ── HTTP GET /snapshot ──→ Backend
  │     └─→ setInitialCanvasState()
  │
  └─→ canvasService.connectWebSocket()  ── WebSocket /ws ──→ Backend
        └─→ setWsConnected(true)

User clicks/drags on Canvas
  │
  ├─→ Update local canvasState (immediate feedback)
  └─→ canvasService.sendPixelUpdate()   ── WebSocket send ──→ Backend
        └─→ Backend broadcasts to all clients
              └─→ canvasService.onPixelUpdate callback
                    └─→ Update canvasState (remote changes)
```

## Running

From the `web/` directory:

```bash
npm install    # Install dependencies (first time only)
npm start      # Start development server on localhost:3000
```

The backend must be running on port 8080 for the frontend to function. See the root README for full setup instructions.
