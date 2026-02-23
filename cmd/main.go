package main

import (
	"log"
	"net/http"
	"os"

	"github.com/AhsanWritesCode/559-project/internal/canvas"
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
	repl = replication.NewReplicator(canvas, os.Getenv("PEERS"), wsHandler)
	wsHandler = server.NewWSHandler(canvas, repl)

	// Handle the HTTP requests, WebSocket connections, and replication
	http.HandleFunc("/snapshot", httpHandler.GetSnapshot)
	http.HandleFunc("/ws", wsHandler.HandleWS)

	http.HandleFunc("/internal/replicate/pixel", repl.HandleReplicatePixel)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
