package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

/*
ElectionMsg is the message sent during an election.
Each node sends its credentials so other nodes can compare and decide
whether to "bully" (override) or accept the sender as a candidate.

Fields:
  - NodeID: the ID of the node starting or participating in the election
  - SnapshotTimestamp: when this node last took a snapshot of its canvas,
    nodes with more recent snapshots are preferred as leaders because they have
    the most up to date data. If timestamps are equal, higher NodeID wins.
*/
type ElectionMsg struct {
	NodeID            int   `json:"node_id"`
	SnapshotTimestamp int64 `json:"snapshot_timestamp"`
}

/*
ElectionResponse is the reply a node sends after receiving an ElectionMsg.

Fields:
  - Bully: true means "I have better credentials than you, back off",
    false means "you are a valid candidate, I won't challenge you"
*/
type ElectionResponse struct {
	Bully bool `json:"bully"`
}

/*
LeaderMsg is broadcast by the new leader to all peers after winning an election.

Fields:
- LeaderID: the new leader's node ID
- LeaderAddr: the new leader's HTTP address so followers know where to forward writes
*/
type LeaderMsg struct {
	LeaderID   int    `json:"leader_id"`
	LeaderAddr string `json:"leader_addr"`
}

/*
ElectionState tracks election specific state on each node.
Kept separate from the main Node mutex to avoid deadlocks during election
since election involves network calls that could block.

Fields:
- mu: protects election state from concurrent access
- inProgress: true while this node is running an election, prevents duplicate elections
- electionTimeout: how long to wait for bully responses before declaring victory
*/
type ElectionState struct {
	mu              sync.Mutex
	inProgress      bool
	electionTimeout time.Duration
}

/*
PeerInfo stores a peer's ID and address together.
Needed for elections so we can map a leader ID to an address
when a new leader is elected.
*/
type PeerInfo struct {
	ID   int
	Addr string
}

/*
initElection sets up the election state on the node.
Called once during node startup.

Inputs:
- timeout: how long to wait for bully responses during an election
*/
func (n *Node) initElection(timeout time.Duration) {
	n.election = &ElectionState{
		electionTimeout: timeout,
	}
}

/*
StartElection begins the Modified Bully election algorithm.
Called when the watchLoop detects that the leader has failed (missed heartbeats).

Algorithm (Modified Bully):
1. Send ElectionMsg to ALL peers with our (snapshotTimestamp, nodeID)
2. Each peer compares our credentials with theirs and responds bully or ok
3. If we receive ANY bully response, back off and wait for a LeaderMsg
4. If NO bully responses and we have quorum (majority), we win
5. Winner sends LeaderMsg to all peers announcing itself as the new leader

Why modified: standard Bully always picks highest ID, which means a node
that just recovered with stale data could become leader. Our version prefers
the node with the most recent snapshot, falling back to node ID as tiebreaker.
*/
func (n *Node) StartElection() {
	n.election.mu.Lock()
	if n.election.inProgress {
		n.election.mu.Unlock()
		log.Printf("[election] node %d ignored StartElection because an election is already in progress", n.NodeID())
		return
	}
	n.election.inProgress = true
	n.election.mu.Unlock()

	defer func() {
		n.election.mu.Lock()
		n.election.inProgress = false
		n.election.mu.Unlock()
	}()

	myID := n.NodeID()
	peers := n.Peers()

	n.mu.RLock()
	myTimestamp := n.snapshotTimestamp
	myAddr := n.selfAddr
	n.mu.RUnlock()

	log.Printf("[election] node %d starting election: self_addr=%s snapshot_ts=%d peers=%d quorum_needed=%d",
		myID, myAddr, myTimestamp, len(peers), ((len(peers)+1)/2)+1)

	payload, _ := json.Marshal(ElectionMsg{
		NodeID:            myID,
		SnapshotTimestamp: myTimestamp,
	})

	// Send election message to all peers concurrently and collect responses
	var mu sync.Mutex
	bullyCount := 0
	okCount := 0

	log.Printf("[election] node %d broadcasting election request to peers", myID)
	var wg sync.WaitGroup
	for _, peer := range peers {
		wg.Add(1)
		go func(peer string) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), n.election.electionTimeout)
			defer cancel()

			url := "http://" + peer + "/internal/election"
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")

			log.Printf("[election] node %d sending election request to peer %s (snapshot_ts=%d)", myID, peer, myTimestamp)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("[election] node %d could not reach peer %s during election: %v", myID, peer, err)
				return
			}
			defer resp.Body.Close()

			var reply ElectionResponse
			if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
				return
			}
			log.Printf("[election] node %d received election response from peer %s: bully=%t", myID, peer, reply.Bully)

			mu.Lock()
			if reply.Bully {
				bullyCount++
			} else {
				okCount++
			}
			mu.Unlock()
		}(peer)
	}

	wg.Wait()
	log.Printf("[election] node %d election round complete: bully_responses=%d ok_responses=%d total_votes=%d", myID, bullyCount, okCount, okCount+1)

	// If any peer bullied us, back off and wait for their leader announcement
	if bullyCount > 0 {
		log.Printf("[election] node %d lost election round: bullied_by=%d peer(s), backing off and waiting for leader announcement", myID, bullyCount)
		return
	}

	// Check quorum: we need a majority of the total cluster (peers + self) to agree
	// We count ourselves as one vote, so okCount + 1 must meet quorum
	totalNodes := len(peers) + 1
	quorum := (totalNodes / 2) + 1

	if okCount+1 >= quorum {
		log.Printf("[election] node %d won election: votes=%d quorum=%d self_addr=%s, transitioning to leader and broadcasting announcement", myID, okCount+1, quorum, myAddr)
		n.becomeLeader(myAddr)
	} else {
		log.Printf("[election] node %d could not become leader: votes=%d required=%d, staying follower", myID, okCount+1, quorum)
	}
}

/*
becomeLeader sets this node as leader and announces to all peers.

Inputs:
- selfAddr: this node's HTTP address, sent to peers so they know where to forward writes

Functions:
- Updates local state to mark this node as leader
- Sends a LeaderMsg to all peers via HTTP POST to /internal/leader
*/
func (n *Node) becomeLeader(selfAddr string) {
	myID := n.NodeID()
	n.SetLeader(myID, selfAddr)

	log.Printf("[election] node %d became the LEADER: leader_addr=%s snapshot_ts=%d", myID, selfAddr, n.SnapshotTimestamp())

	payload, _ := json.Marshal(LeaderMsg{
		LeaderID:   myID,
		LeaderAddr: selfAddr,
	})

	peers := n.Peers()
	for _, peer := range peers {
		go func(peer string) {
			ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
			defer cancel()

			url := "http://" + peer + "/internal/leader"
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")

			log.Printf("[election] node %d sending leadership announcement to peer %s", myID, peer)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("[election] node %d failed to announce leadership to peer %s: %v", myID, peer, err)
				return
			}
			resp.Body.Close()
			log.Printf("[election] node %d successfully announced leadership to peer %s", myID, peer)
		}(peer)
	}
}

/*
HandleElection is the HTTP endpoint that receives election messages from other nodes.
POST /internal/election

Compares the sender's credentials with ours using Modified Bully logic:
1. More recent snapshot wins
2. If same snapshot timestamp, higher node ID wins

If we bully them, we also start our own election since we know
we have better credentials and should be competing.
*/
func (n *Node) HandleElection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var msg ElectionMsg
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	myID := n.NodeID()

	n.mu.RLock()
	myTimestamp := n.snapshotTimestamp
	n.mu.RUnlock()

	log.Printf("[election] node %d received election request from node %d: candidate_snapshot_ts=%d local_snapshot_ts=%d", myID, msg.NodeID, msg.SnapshotTimestamp, myTimestamp)

	// Modified Bully comparison: snapshot timestamp first, then node ID as tiebreaker
	iAmBetter := false
	if myTimestamp > msg.SnapshotTimestamp {
		iAmBetter = true
	} else if myTimestamp == msg.SnapshotTimestamp && myID > msg.NodeID {
		iAmBetter = true
	}
	log.Printf("[election] node %d comparison result against node %d: bully=%t (local_ts=%d peer_ts=%d local_id=%d peer_id=%d)", myID, msg.NodeID, iAmBetter, myTimestamp, msg.SnapshotTimestamp, myID, msg.NodeID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ElectionResponse{Bully: iAmBetter})

	if iAmBetter {
		if n.IsLeader() {
			log.Printf("[election] node %d ignored election trigger from node %d because it is already leader", myID, msg.NodeID)
			return
		}

		leaderID := n.LeaderID()
		if leaderID != 0 {
			log.Printf("[election] node %d suppressed election trigger from node %d because leader is already known as Node %d", myID, msg.NodeID, leaderID)
			return
		}

		log.Printf("[election] node %d bullied node %d because it has better election credentials, knows no leader, starting its own election", myID, msg.NodeID)
		go n.StartElection()
	}
}

/*
HandleLeaderAnnouncement is the HTTP endpoint that receives the new leader announcement.
POST /internal/leader

When a node wins an election it sends a LeaderMsg to all peers.
We update our local state to point to the new leader and reset heartbeat state
so we don't immediately trigger another election.
*/
func (n *Node) HandleLeaderAnnouncement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var msg LeaderMsg
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	oldLeaderID := n.LeaderID()
	n.SetLeader(msg.LeaderID, msg.LeaderAddr)
	log.Printf("[election] node %d accepted new leader announcement: old_leader=%d new_leader=%d leader_addr=%s", n.NodeID(), oldLeaderID, msg.LeaderID, msg.LeaderAddr)

	// Stop any in progress election since a leader has been chosen
	n.election.mu.Lock()
	n.election.inProgress = false
	n.election.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

/*
SetSnapshotTimestamp updates the snapshot timestamp for this node.
Called whenever a snapshot is successfully saved to disk.
Used during elections to determine which node has the most recent data.

Inputs:
- ts: Unix timestamp (seconds) of when the snapshot was taken
*/
func (n *Node) SetSnapshotTimestamp(ts int64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.snapshotTimestamp = ts
}

/*
SnapshotTimestamp returns the current snapshot timestamp for this node.
*/
func (n *Node) SnapshotTimestamp() int64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.snapshotTimestamp
}

/*
ParsePeersWithIDs parses the PEERS env var in "id=address" format.
Example: "1=localhost:8080,2=localhost:8081,3=localhost:8082"

Returns:
- []PeerInfo: parsed peer info with IDs and addresses
- []string: just the addresses for existing code that uses plain address lists
*/
func ParsePeersWithIDs(peersCSV string) ([]PeerInfo, []string) {
	var infos []PeerInfo
	var addrs []string

	for _, p := range strings.Split(peersCSV, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var info PeerInfo
		if n, err := fmt.Sscanf(p, "%d=%s", &info.ID, &info.Addr); n == 2 && err == nil {
			infos = append(infos, info)
			addrs = append(addrs, info.Addr)
		} else {
			// Fallback: plain address with no ID
			addrs = append(addrs, p)
		}
	}

	return infos, addrs
}
