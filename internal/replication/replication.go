package replication

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/AhsanWritesCode/559-project/internal/canvas"
	"github.com/AhsanWritesCode/559-project/internal/models"
)

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
- leaderAddr: string linking the follower nodes to the leader
*/
type Replicator struct {
	canvas      *canvas.Canvas
	peers       []string
	httpClient  *http.Client
	broadcaster Broadcaster
	leaderAddr  string
}

/*
THis is a new replicator struct
- c: pointer to shared canvas
- peersCSV: comma-separated list of peer addresses
- b: broadcaster for local WebSocket clients
- leaderAddr: address for leader

Functions:
- It gets the peer addresses.
- It creates httpClient with timeout
- It stores broadcaster for optional local notifications

Returns a new functioning replicator
*/
func NewReplicator(c *canvas.Canvas, peersCSV string, leaderAddr string, b Broadcaster) *Replicator {
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
		leaderAddr:  strings.TrimSpace(leaderAddr),
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

/*
This is a function to forward pixel updates to the leader.

Inputs:
- rawMsg: pixel update in JSON

Functions:
- It sends an HTTP POST request to the leader's /internal/forward/pixel endpoint
- Uses a timeout context to avoid hanging requests
- Returns an error if the request fails or if the leader responds with a non-2xx status
- Does nothing if no leader is set i.e. that node is the leader.
*/
func (r *Replicator) ForwardToLeader(rawMsg []byte) error {
	if r.leaderAddr == "" {
		return nil // do nothing
	}

	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()

	url := "http://" + r.leaderAddr + "/internal/forward/pixel"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawMsg))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &httpError{status: resp.Status}
	}
	return nil
}

/*
Represents an HTTP error returned from a request that failed.

Fields:
- status: the HTTP response status string

Functions:
  - Implements the error interface so HTTP failures can be returned and handled like regular
    Go errors
*/
type httpError struct{ status string }

func (e *httpError) Error() string { return e.status }

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
	var upd models.PixelUpdate
	if err := json.NewDecoder(req.Body).Decode(&upd); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	if ok := r.canvas.SetPixel(upd.X, upd.Y, upd.Color); !ok {
		http.Error(w, "out of bounds", http.StatusBadRequest)
		return
	}

	log.Printf("[follower] applied replicated pixel x=%d y=%d color=%s", upd.X, upd.Y, upd.Color)

	// Broadcast to local clients
	if r.broadcaster != nil {
		raw, _ := json.Marshal(upd)
		r.broadcaster.BroadcastRaw(raw)
	}

	w.WriteHeader(http.StatusOK)
}

/*
This is a leader endpoint.
This handles incoming pixel updates from clients on the leader.

Inputs:
- w: HTTP response writer
- req: HTTP request containing the pixel update in JSON

Functions:
- Checks that the request method is POST
- Decodes the JSON into a PixelUpdate
- Applies the pixel change to the local canvas
- Replicates the update to all follower peers
- Broadcasts the update to the leader's local WebSocket clients
- Returns HTTP 200 OK if successful

Purpose: allows leader to send updates to followers
*/
func (r *Replicator) HandleForwardPixel(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer req.Body.Close()

	rawBytes, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}

	if err := r.CommitPixelRaw(rawBytes); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

/*
*
This commits a pixel update to the local canvas and propagates it.
Made it its own function so that I can use it if updates are send directly to the leader
or if they are coming from a follower.

Inputs:
- msg: a JSON-encoded PixelUpdate as a byte slice

Functions:
- Decodes the JSON into a PixelUpdate struct
- Applies the pixel change to the local canvas
- Returns an HTTP-style error if the update is out of bounds
- Replicates the update to all follower peers
- Broadcasts the update to local WebSocket clients if a broadcaster is set
- Returns nil if the update is successful
*/
func (r *Replicator) CommitPixelRaw(msg []byte) error {
	var upd models.PixelUpdate
	if err := json.Unmarshal(msg, &upd); err != nil {
		return err
	}
	if ok := r.canvas.SetPixel(upd.X, upd.Y, upd.Color); !ok {
		return fmt.Errorf("out of bounds")
	}

	log.Printf("[leader] committed pixel x=%d y=%d color=%s — replicating to %d peers", upd.X, upd.Y, upd.Color, len(r.peers))

	// replicate to peers + broadcast to clients
	r.ReplicateToPeers(msg)
	if r.broadcaster != nil {
		r.broadcaster.BroadcastRaw(msg)
	}
	return nil
}
