package node

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
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
HTTP endpoint followers use to receive leader heartbeats.

Inputs:
- n: Current replica's instance of nod
- w: HTTP response writer
- r: HTTP request containing HeartbeatMsg in JSON

Functions:
- It ensures request method is POST
- It decodes JSON into HeartbeatMsg
- It updates lastHeartbeat, leaderID, and leaderAlive thread-safely
- It returns HTTP 200 OK if successful, 400/405 on errors
*/
func (n *Node) HandleHeartbeat(w http.ResponseWriter, r *http.Request) {
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

	n.UpdateHeartbeatFromLeader(msg.LeaderID)
	w.WriteHeader(http.StatusOK)
}

/*
Start runs the heartbeat loops for the node.

Inputs:
- n: current instance of replica's node

Functions:
- Starts leaderLoop if this node is leader (sends heartbeats periodically)
- Starts watchdogLoop if this node is follower (checks for timeout)
*/
func (n *Node) StartHeartbeats(interval, timeout time.Duration) {
	go n.leaderLoop(interval)
	go n.watchLoop(timeout)
}

/*
Runs in a goroutine to send heartbeats periodically if this node is leader.

Functions:
- It uses a ticker with interval h.interval
- On each tick, sends heartbeat to all peers

Citations:
- I learnt about tickers here: https://dev.to/ankitmalikg/go-ticker-vs-timer-4glb
*/
func (n *Node) leaderLoop(interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for range t.C {
		if !n.IsLeader() {
			continue
		}
		n.sendHeartbeatToPeers()
	}
}

/*
Sends a heartbeat message to all peers.

Functions:
- It decodes heartbeat messahes
- It loops through peers, sending HTTP POST to /internal/heartbeat
- Uses goroutines for concurrency
*/
func (n *Node) sendHeartbeatToPeers() {
	leaderID := n.LeaderID()
	peers := n.Peers()

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

			resp, err := http.DefaultClient.Do(req)
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
func (n *Node) watchLoop(timeout time.Duration) {
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()

	for range t.C {
		if n.IsLeader() {
			continue
		}

		n.mu.Lock()
		since := time.Since(n.lastHeartbeat)
		alive := n.leaderAlive
		n.mu.Unlock()
		if since > timeout && alive {
			n.MarkLeaderDead()
			log.Printf("heartbeat: leader %d missed for %v (timeout %v). Leader may be dead.",
				n.LeaderID(), since.Truncate(time.Millisecond), timeout)
		}
	}
}
