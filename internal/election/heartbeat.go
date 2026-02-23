package election

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

/*
This is basically a heartbeat sent from the leader to followers.

Inputs:
- LeaderID: the ID of the current leader
- Term: it is for the raft part. Still working on that
*/
type HeartbeatMsg struct {
	LeaderID int `json:"leader_id"`
	Term     int `json:"term,omitempty"`
}

/*
Heartbeats manages leader heartbeat sending and follower monitoring.

Inputs:
- nodeID: this node’s ID
- leaderID: current leader ID
- peers: list of peer addresses
- lastHeartbeat: timestamp of last received heartbeat
- leaderAlive: whether leader is considered alive
- interval: heartbeat send interval
- timeout: follower timeout to suspect leader failure
- httpClient: HTTP client for sending heartbeats
*/
type Heartbeats struct {
	mu sync.Mutex

	nodeID   int
	leaderID int
	peers    []string

	lastHeartbeat time.Time
	leaderAlive   bool

	interval time.Duration
	timeout  time.Duration

	httpClient *http.Client
}

/*
*
NewHeartbeats creates a Heartbeats instance for a node.

Inputs:
- nodeID: ID of this node
- leaderID: ID of initial leader
- peersCSV: comma-separated list of peer addresses
- interval: heartbeat send interval
- timeout: follower heartbeat timeout

Functions:
- It identifies the peers
- It initializes lastHeartbeat to now and leaderAlive to true
- It returns a Heartbeats struct ready to start loops
*/
func NewHeartbeats(nodeID, leaderID int, peersCSV string, interval, timeout time.Duration) *Heartbeats {
	peers := []string{}
	for _, p := range strings.Split(peersCSV, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			peers = append(peers, p)
		}
	}

	return &Heartbeats{
		nodeID:   nodeID,
		leaderID: leaderID,
		peers:    peers,

		lastHeartbeat: time.Now(),
		leaderAlive:   true,

		interval: interval,
		timeout:  timeout,

		httpClient: &http.Client{Timeout: 800 * time.Millisecond},
	}
}

// Returns true if this node is currently the leader.
func (h *Heartbeats) IsLeader() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.nodeID == h.leaderID
}

// Returns whether the leader is currently considered alive
func (h *Heartbeats) LeaderAlive() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.leaderAlive
}

// Updates the leader ID and resets heartbeat state.
func (h *Heartbeats) SetLeaderID(leaderID int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.leaderID = leaderID
	// reset alive view when leader changes
	h.leaderAlive = true
	h.lastHeartbeat = time.Now()
}

/*
HTTP endpoint followers use to receive leader heartbeats.

Inputs:
- w: HTTP response writer
- r: HTTP request containing HeartbeatMsg in JSON

Functions:
- It ensures request method is POST
- It decodes JSON into HeartbeatMsg
- It updates lastHeartbeat, leaderID, and leaderAlive thread-safely
- It returns HTTP 200 OK if successful, 400/405 on errors
*/
func (h *Heartbeats) HandleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var msg HeartbeatMsg
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	h.leaderID = msg.LeaderID
	h.lastHeartbeat = time.Now()
	h.leaderAlive = true
	h.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

/*
Start runs the heartbeat loops for the node.

Functions:
- Starts leaderLoop if this node is leader (sends heartbeats periodically)
- Starts watchdogLoop if this node is follower (checks for timeout)
*/
func (h *Heartbeats) Start() {
	go h.leaderLoop()
	go h.watchLoop()
}

/*
Runs in a goroutine to send heartbeats periodically if this node is leader.

Functions:
- It uses a ticker with interval h.interval
- On each tick, sends heartbeat to all peers

Citations:
- I learnt about tickers here: https://dev.to/ankitmalikg/go-ticker-vs-timer-4glb
*/
func (h *Heartbeats) leaderLoop() {
	t := time.NewTicker(h.interval)
	defer t.Stop()

	for range t.C {
		if !h.IsLeader() {
			continue
		}
		h.sendHeartbeatToPeers()
	}
}

/*
Sends a heartbeat message to all peers.

Functions:
- It decodes heartbeat messahes
- It loops through peers, sending HTTP POST to /internal/heartbeat
- Uses goroutines for concurrency
*/
func (h *Heartbeats) sendHeartbeatToPeers() {
	h.mu.Lock()
	leaderID := h.leaderID
	peers := append([]string{}, h.peers...)
	h.mu.Unlock()

	payload, _ := json.Marshal(HeartbeatMsg{LeaderID: leaderID})

	for _, peer := range peers {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
			defer cancel()

			url := "http://" + peer + "/internal/heartbeat"
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
			if err != nil {
				log.Println("heartbeat: build request error:", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := h.httpClient.Do(req)
			if err != nil {
				// follower might be down. That is not the leaders problem.
				return
			}
			_ = resp.Body.Close()
		}()
	}
}

/*
Runs in a goroutine for followers to detect leader failure.

Functions:
- It uses a ticker (currently set at 200ms) to check time since last heartbeat
- If timeout exceeded, marks leaderAlive = false
- Logs that leader is suspected dead (TODO Call for an election in future updates)
- Does nothing if this node is the leader
*/
func (h *Heartbeats) watchLoop() {
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()

	for range t.C {
		if h.IsLeader() {
			continue
		}

		h.mu.Lock()
		since := time.Since(h.lastHeartbeat)
		if since > h.timeout && h.leaderAlive {
			h.leaderAlive = false
			log.Printf("heartbeat: leader %d missed for %v (timeout %v). Leader may be dead.",
				h.leaderID, since.Truncate(time.Millisecond), h.timeout)
		}
		h.mu.Unlock()
	}
}
