package main

import (
	"log"
	"net/http"

	"github.com/AhsanWritesCode/559-project/internal/canvas"
	"github.com/AhsanWritesCode/559-project/internal/server"
)

// entry point for the application, initializes the canvas and the HTTP and Websocket handlers and starts the server
func main() {
	canvas := canvas.NewCanvas(canvas.DefaultCanvasWdith, canvas.DefaultCanvasHeight)

	//Create the HTTP and Websocket handlers here
	httpHandler := server.NewHttpHandler(canvas)
	wsHandler := server.NewWSHandler(canvas)

	// Handle the HTTP requests and WebSocket connections
	http.HandleFunc("/snapshot", httpHandler.GetSnapshot)
	http.HandleFunc("/ws", wsHandler.HandleWS)

	// Handle the status endpoint for load balancing
	http.HandleFunc("/status", wsHandler.ClientCount)

	log.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
