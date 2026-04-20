package replication

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/AhsanWritesCode/559-project/internal/models"
	"github.com/AhsanWritesCode/559-project/internal/node"
)

// Small function to show crashing while replicating
func shouldCrashOnReplicationFlag(nodeID int) bool {
	flag := fmt.Sprintf("crash_node_%d_on_replication.flag", nodeID)
	_, err := os.Stat(flag)
	if err == nil {
		_ = os.Remove(flag)
		return true
	}
	return false
}

/*
This function is hwo the leader replicates to others after applying changes locally
- Node: replica this is acting on
- rawMsg: original JSON message from client

Functions:
- It loops over all peer servers
- It sends each peer a POST request with the raw pixel update
- It uses goroutines for replication ("best effort" so leader isn't blocked forever)
- Uses timeout to avoid blocking leader forever

- It logs errors but does not crash if a peer is unreachable

Leader uses this to push updates to all peer/follower servers
*/
func ReplicateToPeers(n *node.Node, rawMsg []byte) {
	peers := n.Peers()

	for _, peer := range peers {
		go func(peer string) {
			// Just to crash stuff
			if shouldCrashOnReplicationFlag(n.NodeID()) {
				log.Printf("replication: crash flag detected for node %d, crashing node", n.NodeID())
				os.Exit(1)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
			defer cancel()

			url := "http://" + peer + "/internal/replicate/pixel"
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawMsg))
			if err != nil {
				log.Println("replication: build request error:", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")

			// this is best effort for now. I will check and see if we make any guarantees later
			resp, err := http.DefaultClient.Do(req)
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
- Node: replica this is acting on

Functions:
- It sends an HTTP POST request to the leader's /internal/forward/pixel endpoint
- Uses a timeout context to avoid hanging requests
- Returns an error if the request fails or if the leader responds with a non-2xx status
- Does nothing if no leader is set i.e. that node is the leader.
*/
func ForwardToLeader(n *node.Node, rawMsg []byte) error {
	leaderAddr := n.LeaderAddr()
	if leaderAddr == "" {
		return nil // do nothing
	}

	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()

	url := "http://" + leaderAddr + "/internal/forward/pixel"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawMsg))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// still best efforting a bit
	resp, err := http.DefaultClient.Do(req)
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
- n: node instance for this replica

Functions:
- It checks request method is POST
- Then it decodes incoming JSON to PixelUpdate
- Then it applies pixel change to local canvas
- It broadcasts update to local WS clients if broadcaster is set
- Returns HTTP status OK (200) if successful (i.e all good!)

Purpose: allows followers to receive replicated updates from the leader
*/
func HandleReplicatePixel(n *node.Node, w http.ResponseWriter, req *http.Request) {
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

	if ok := n.Canvas.SetPixel(upd.X, upd.Y, upd.Color); !ok {
		http.Error(w, "out of bounds", http.StatusBadRequest)
		return
	}

	log.Printf("[follower] applied replicated pixel x=%d y=%d color=%s", upd.X, upd.Y, upd.Color)

	// Tell node to broadcast to local clients
	raw, _ := json.Marshal(upd)
	n.BroadcastRaw(raw)
	w.WriteHeader(http.StatusOK)
}

/*
This is a leader endpoint.
This handles incoming pixel updates from clients on the leader.

Inputs:
- w: HTTP response writer
- req: HTTP request containing the pixel update in JSON
- n: local instance of node

Functions:
- Checks that the request method is POST
- Decodes the JSON into a PixelUpdate
- Applies the pixel change to the local canvas
- Tells it to commit the pixels to followers
- Returns HTTP 200 OK if successful

Purpose: allows leader to send updates to followers
*/
func HandleForwardPixel(n *node.Node, w http.ResponseWriter, req *http.Request) {
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

	if err := CommitPixelRaw(n, rawBytes); err != nil {
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
func CommitPixelRaw(n *node.Node, msg []byte) error {
	var upd models.PixelUpdate
	if err := json.Unmarshal(msg, &upd); err != nil {
		return err
	}
	if ok := n.Canvas.SetPixel(upd.X, upd.Y, upd.Color); !ok {
		return fmt.Errorf("out of bounds")
	}

	log.Printf("[leader] committed pixel x=%d y=%d color=%s — replicating to %d peers", upd.X, upd.Y, upd.Color, len(n.Peers()))

	// replicate to peers + broadcast to clients
	ReplicateToPeers(n, msg)
	n.BroadcastRaw(msg)
	return nil
}
