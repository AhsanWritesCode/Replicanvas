package server

import (
	"log"
	"net/http"
	"sync"

	"github.com/AhsanWritesCode/559-project/internal/node"
	"github.com/AhsanWritesCode/559-project/internal/replication"
	"github.com/gorilla/websocket"
)

// This upgrader is used to upgrade the HTTP connection to a WebSocket connection
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// This is the WebSocket handler struct
// It contains the canvas, a mutex to synchronize access to the clients map, and a map of connected clients
/*
- replicator: if it is the leader, this is how it communicates
- nodeID: identification
- leaderID: who does it think is the leader right now
*/
type WSHandler struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
	node    *node.Node
}

/*
This is the WebSocket handler constructor

Inputs:
- canvas: shared canvas to read and update pixels
- node instance

Functions:
- It creates a new WSHandler instance
- Initializes clients map to track connected WebSocket clients
- Stores the replicator so pixel updates can be sent to peers if needed
- Reads NODE_ID and LEADER_ID from environment variables

Purpose: sets up a WebSocket handler that can manage connected clients and optionally

	replicate updates across nodes.
*/
func NewWSHandler(n *node.Node) *WSHandler {
	return &WSHandler{
		clients: make(map[*websocket.Conn]bool),
		node:    n,
	}
}

/*
This is a Broadcaster interface.

Inputs:
- msg: a byte message that needs to be sent to all other users

Functions:
- It allows other packages can use broadcasting from outside the server package.
*/
func (ws *WSHandler) BroadcastRaw(msg []byte) {
	ws.broadcast(msg)
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
	//read message from the clients and apply the pixel update to the canvas and broadcast
	// the update to all connected clients
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// follower: forward only
		if !ws.node.IsLeader() {
			if err := replication.ForwardToLeader(ws.node, msg); err != nil {
				log.Println("forward to leader failed:", err)
			}
			continue
		}

		// leader: commit using shared function
		if err := replication.CommitPixelRaw(ws.node, msg); err != nil {
			log.Println("commit failed:", err)
		}
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
