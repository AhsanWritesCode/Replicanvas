package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/AhsanWritesCode/559-project/internal/canvas"
	"github.com/gorilla/websocket"
)

// This upgrader is used to upgrade the HTTP connection to a WebSocket connection
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// This is the message format between the server and the client for pixel updates
type PixelUpdate struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Color string `json:"color"`
}

// This is the WebSocket handler struct
// It contains the canvas, a mutex to synchronize access to the clients map, and a map of connected clients
type WSHandler struct {
	canvas  *canvas.Canvas
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
}

// This function is used to create a new WebSocket handler for the canvas
func NewWSHandler(canvas *canvas.Canvas) *WSHandler {
	return &WSHandler{canvas: canvas, clients: make(map[*websocket.Conn]bool)}
}

// Function to handle the WebSocket connection
// We use the sync.Mutex to synchronize access to the clients map
// This is because multiple goroutines can access the clients map concurrently,
// which can cause race conditions and potentially corrupt the data
// E.g. if a new client connects and we add it to the clients map thats a goroutine, and if another client disconnects and we remove it from the clients map thats another goroutine,
// thats 2 goroutines access the clients map concurrently, without a mutex, the data will run into a race conditions, Go will panic or corrupt memory
func (ws *WSHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	//add the client to the clients map
	//same reason as above for the mutex use here
	ws.mu.Lock()
	ws.clients[conn] = true
	ws.mu.Unlock()

	log.Printf("Client connected (%d total)", len(ws.clients))

	//This defer function is used to clean up the client connection and remove it from the clients map
	//and close the connection
	defer func() {
		ws.mu.Lock()
		delete(ws.clients, conn)
		ws.mu.Unlock()
		conn.Close()
		log.Printf("The Client disconnected (%d total)", len(ws.clients))
	}()

	//main loop to handle the websocket connection
	//read message from the clients and apply the pixel update to the canvas and broadcast the update to all connected clients
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		//unmarshal the message into the PixelUpdate struct
		//if the message is invalid, log the error and continue
		var update PixelUpdate
		if err := json.Unmarshal(msg, &update); err != nil {
			log.Println("Invalid message:", err)
			continue
		}

		// Apply the pixel update to the canvas
		if !ws.canvas.SetPixel(update.X, update.Y, update.Color) {
			log.Printf("Out of bounds: (%d, %d)", update.X, update.Y)
			continue
		}

		// Broadcast to all connected clients
		ws.broadcast(msg)
	}
}

// Function to broadcast the pixels update
func (ws *WSHandler) broadcast(msg []byte) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	//iterate over the clients map (connected clients) and send the message to each client
	for conn := range ws.clients {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			conn.Close()
			delete(ws.clients, conn)
		}
	}
}

// response shape for the /status endpoint
type StatusResponse struct {
	Connections int `json:"connections"`
}

// returns the number of connected clients so the frontend can pick the least-loaded server
func (ws *WSHandler) ClientCount(w http.ResponseWriter, r *http.Request) {
	ws.mu.Lock()
	count := len(ws.clients)
	ws.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(StatusResponse{Connections: count})
}
