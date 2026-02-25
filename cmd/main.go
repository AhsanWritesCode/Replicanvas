package main

import (
	"log"
	"net/http"
	"os"

	"github.com/AhsanWritesCode/559-project/internal/canvas"
	"github.com/AhsanWritesCode/559-project/internal/config"
	"github.com/AhsanWritesCode/559-project/internal/election"
	"github.com/AhsanWritesCode/559-project/internal/replication"
	"github.com/AhsanWritesCode/559-project/internal/server"
)

// entry point for the application, initializes the canvas and the HTTP and Websocket handlers and starts the server
func main() {
	canvas := canvas.NewCanvas(canvas.DefaultCanvasWdith, canvas.DefaultCanvasHeight)

	//Create the HTTP and Websocket handlers here
	httpHandler := server.NewHttpHandler(canvas)
	var wsHandler *server.WSHandler
	var repl *replication.Replicator

	wsHandler = server.NewWSHandler(canvas, nil)
	repl = replication.NewReplicator(canvas, os.Getenv("PEERS"), os.Getenv("LEADER_ADDR"), wsHandler)
	wsHandler.SetReplicator(repl)

	// Handle the HTTP requests, WebSocket connections, and replication
	http.HandleFunc("/snapshot", httpHandler.GetSnapshot)
	http.HandleFunc("/ws", wsHandler.HandleWS)
	http.HandleFunc("/internal/forward/pixel", repl.HandleForwardPixel)
	http.HandleFunc("/internal/replicate/pixel", repl.HandleReplicatePixel)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// TODO: Properly implement this stuff later.
	// FROM >>>>>>>>
	nodeID := config.MustIntEnv("NODE_ID", 1)
	leaderID := config.MustIntEnv("LEADER_ID", 1)

	hbInterval := config.MustDurationEnvMs("HB_INTERVAL_MS", 500)
	hbTimeout := config.MustDurationEnvMs("HB_TIMEOUT_MS", 1500)

	hb := election.NewHeartbeats(nodeID, leaderID, os.Getenv("PEERS"), hbInterval, hbTimeout)

	// internal endpoint to receive heartbeats
	http.HandleFunc("/internal/heartbeat", hb.HandleHeartbeat)

	// start loops
	hb.Start()
	// TO <<<<<<<

	log.Println("Server running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
