package server

import (
	"encoding/json"
	"net/http"

	"github.com/AhsanWritesCode/559-project/internal/node"
)

// This struct is used to handle the HTTP requests and responses for the canvas
type HttpHandler struct {
	node *node.Node
}

// This function is used to create a new HTTP handler for the canvas
func NewHttpHandler(node *node.Node) *HttpHandler {
	return &HttpHandler{node: node}
}

// This struct is used to represent the response for the snapshot of the canvas (it will be sent to the client to be displayed on the canvas)
type SnapShotResponse struct {
	Width  int        `json:"width"`
	Height int        `json:"height"`
	Pixels [][]string `json:"pixels"`
}

// This request is used to get a snapshot of the canvas (it will be used when a client first connects to the server, or when a node crashes and reconnects to the server and wants to get the latest snapshot of the canvas)
// This is not currently checking for any errors or validation of the request, in fact its not even checking if the request is a GET request (we are assuming its a GET request for now)
// The frontend that is calling this will use GET request to get the snapshot of the canvas for now (we'll add validation and error checking later)
func (h *HttpHandler) GetSnapshot(w http.ResponseWriter, r *http.Request) {

	//content type and access control allow origin headers to allow the client to access the snapshot of the canvas
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	response := SnapShotResponse{
		Width:  h.node.Canvas.Width,
		Height: h.node.Canvas.Height,
		Pixels: h.node.Canvas.Snapshot(),
	}
	json.NewEncoder(w).Encode(response)
}
