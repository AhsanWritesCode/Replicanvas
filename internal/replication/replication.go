package replication

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/AhsanWritesCode/559-project/internal/canvas"
)

// PixelUpdate matching the client WS payload
type PixelUpdate struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Color string `json:"color"`
}

// Broadcaster to allow Replication Notify Local WS Clients. I will figure this out once I can
// figure out the web then I will actually start commenting this properly.
type Broadcaster interface {
	BroadcastRaw(msg []byte)
}

/*
THis is a Replicator Struct.
COntains:
- canvas: shared canvas state to update pixels locally
- peers: a list of other servers to send replicas to
- httpClient: connection I will use to send requests/messages to peers
- broadcaster: to notify local WS clients. May include if necessary
*/
type Replicator struct {
	canvas      *canvas.Canvas
	peers       []string
	httpClient  *http.Client
	broadcaster Broadcaster
}

/*
THis is a new replicator struct
- c: pointer to shared canvas
- peersCSV: comma-separated list of peer addresses
- b: optional broadcaster for local WebSocket clients

Functions:
- It gets the peer addresses.
- It creates httpClient with timeout
- It stores broadcaster for optional local notifications

Returns a new functioning replicator
*/
func NewReplicator(c *canvas.Canvas, peersCSV string, b Broadcaster) *Replicator {
	peers := []string{}
	for _, p := range strings.Split(peersCSV, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			peers = append(peers, p)
		}
	}
	return &Replicator{
		canvas: c,
		peers:  peers,
		httpClient: &http.Client{
			Timeout: 800 * time.Millisecond,
		},
		broadcaster: b,
	}
}

/*
This function is hwo the leader replicates to others after applying changes locally
- rawMsg: original JSON message from client

Functions:
- It loops over all peer servers
- It sends each peer a POST request with the raw pixel update
- It uses goroutines for replication ("best effort" so leader isn't blocked forever)
  - Uses timeout to avoid blocking leader forever

- It logs errors but does not crash if a peer is unreachable

Leader uses this to push updates to all peer/follower servers
*/
func (r *Replicator) ReplicateToPeers(rawMsg []byte) {
	for _, peer := range r.peers {
		go func(peer string) {
			ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
			defer cancel()

			url := "http://" + peer + "/internal/replicate/pixel"
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawMsg))
			if err != nil {
				log.Println("replication: build request error:", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := r.httpClient.Do(req)
			if err != nil {
				log.Println("replication: POST failed to", peer, ":", err)
				return
			}
			_ = resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				log.Println("replication: non-2xx from", peer, ":", resp.Status)
			}
		}(peer)
	}
}

// Follower endpoint: POST /internal/replicate/pixel
/*
THis is a follower endpoint,
- HTTP POST endpoint at /internal/replicate/pixel

Functions:
- It checks request method is POST
- Then it decodes incoming JSON to PixelUpdate
- Then it applies pixel change to local canvas
- It broadcasts update to local WS clients if broadcaster is set
- Returns HTTP status OK (200) if successful (i.e all good!)

Purpose: allows followers to receive replicated updates from the leader
*/
func (r *Replicator) HandleReplicatePixel(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer req.Body.Close()

	// We decode, then also re-encode to bytes for local WS broadcast.
	var upd PixelUpdate
	if err := json.NewDecoder(req.Body).Decode(&upd); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	if ok := r.canvas.SetPixel(upd.X, upd.Y, upd.Color); !ok {
		http.Error(w, "out of bounds", http.StatusBadRequest)
		return
	}

	// Broadcast to local clients
	if r.broadcaster != nil {
		raw, _ := json.Marshal(upd)
		r.broadcaster.BroadcastRaw(raw)
	}

	w.WriteHeader(http.StatusOK)
}
