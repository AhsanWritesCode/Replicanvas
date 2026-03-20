package main

import (
	"log"
	"net/http"
	"os"

	"github.com/AhsanWritesCode/559-project/internal/canvas"
	"github.com/AhsanWritesCode/559-project/internal/config"
	"github.com/AhsanWritesCode/559-project/internal/node"
	"github.com/AhsanWritesCode/559-project/internal/replication"
	"github.com/AhsanWritesCode/559-project/internal/server"
)

// entry point for the application, initializes the canvas and the HTTP and Websocket handlers and starts the server
func main() {
	c := canvas.NewCanvas(canvas.DefaultCanvasWdith, canvas.DefaultCanvasHeight)

	n := node.NewNode(
		c,
		config.MustIntEnv("NODE_ID", 1),
		config.MustIntEnv("LEADER_ID", 1),
		os.Getenv("PEERS"),
		os.Getenv("LEADER_ADDR"),
	)

	//Create the HTTP and Websocket handlers here
	httpHandler := server.NewHttpHandler(n)
	var wsHandler *server.WSHandler
	wsHandler = server.NewWSHandler(n)

	n.SetBroadcaster(wsHandler)

	// Handle the HTTP requests, WebSocket connections, and replication
	http.HandleFunc("/snapshot", httpHandler.GetSnapshot)
	http.HandleFunc("/ws", wsHandler.HandleWS)
	http.HandleFunc("/internal/forward/pixel", func(w http.ResponseWriter, r *http.Request) {
		replication.HandleForwardPixel(n, w, r)
	})
	http.HandleFunc("/internal/replicate/pixel", func(w http.ResponseWriter, r *http.Request) {
		replication.HandleReplicatePixel(n, w, r)
	})
	// internal endpoint to receive heartbeats
	http.HandleFunc("/internal/heartbeat", n.HandleHeartbeat)

	// TODO: Properly implement this stuff later.
	// FROM >>>>>>>>
	hbInterval := config.MustDurationEnvMs("HB_INTERVAL_MS", 500)
	hbTimeout := config.MustDurationEnvMs("HB_TIMEOUT_MS", 1500)
	n.StartHeartbeats(hbInterval, hbTimeout)
	// TO <<<<<<<

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
