package node

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/AhsanWritesCode/559-project/internal/canvas"
)

// It is an interface shadowing this function present in websocket.go
// Broadcaster to allow Node to Notify Local WS Clients of changes.
type Broadcaster interface {
	BroadcastRaw(msg []byte)
}

/*
Node is a central tracker for all things related to a replica's
state and state management.

Inputs:
- Canvas: shared canvas state to update pixels locally
- Mu: mutex for safe concurrent access
- NodeID: this node’s ID
- LeaderID: current leader’s ID
- Peers: list of peer addresses
- LeaderAddr: address of the leader
- PeerInfos: list of peers with IDs and addresses (needed for elections to map ID → address)
- SelfAddr: this node's own HTTP address (sent in leader announcements so followers know where to forward writes)
- SnapshotTimestamp: Unix timestamp of last snapshot (used in Modified Bully to prefer nodes with recent data)
- LastHeartbeat: last time a heartbeat was received
- LeaderAlive: whether the leader is alive
- Election: election-specific state (in-progress flag, timeout)
*/
type Node struct {
	Canvas *canvas.Canvas

	mu sync.RWMutex

	nodeID    int
	leaderID  int
	peers     []string
	peerInfos []PeerInfo

	selfAddr          string
	leaderAddr        string
	snapshotTimestamp int64

	lastHeartbeat time.Time
	leaderAlive   bool

	broadcaster Broadcaster
	election    *ElectionState
}

/*
NewNode creates and initializes a new Node.

Inputs:
  - c: shared canvas state
  - nodeID: this node’s ID
  - leaderID: current leader’s ID
  - peersCSV: comma-separated list of peer addresses in "id=address" format
    (e.g. "1=localhost:8080,2=localhost:8081,3=localhost:8082")
  - selfAddr: this node’s own HTTP address
  - leaderAddr: address of the leader
  - electionTimeout: how long to wait for bully responses during an election
*/
func NewNode(c *canvas.Canvas, nodeID, leaderID int, peersCSV, selfAddr, leaderAddr string, electionTimeout time.Duration) *Node {
	peerInfos, peers := ParsePeersWithIDs(peersCSV)

	// Fallback: if peers were provided without IDs (plain addresses), keep them as-is
	if len(peerInfos) == 0 && len(peers) == 0 {
		for _, p := range strings.Split(peersCSV, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				peers = append(peers, p)
			}
		}
	}

	n := &Node{
		Canvas: c,

		nodeID:    nodeID,
		leaderID:  leaderID,
		peers:     peers,
		peerInfos: peerInfos,

		selfAddr:          selfAddr,
		leaderAddr:        leaderAddr,
		snapshotTimestamp: 0,

		lastHeartbeat: time.Now(),
		leaderAlive:   true,
	}

	n.initElection(electionTimeout)

	// On startup, trigger an election after a short delay to allow the HTTP server to start.
	// This ensures that a recovering node doesn't just assume it's the leader from its old condfig, otherwise it will just become the leader again without discovering that there is a new leader
	go func() {
		time.Sleep(2 * time.Second)
		log.Printf("[startup] node %d triggering initial election to discover or become leader", nodeID)
		n.StartElection()
	}()

	return n
}

// returns this node’s ID.
func (n *Node) NodeID() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.nodeID
}

// returns the current leader’s ID.
func (n *Node) LeaderID() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.leaderID
}

// returns the current leader’s address.
func (n *Node) LeaderAddr() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.leaderAddr
}

// returns whether the leader is considered alive.
func (n *Node) LeaderAlive() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.leaderAlive
}

// returns a copy of the peer list.
func (n *Node) Peers() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return append([]string{}, n.peers...)
}

// checks if this node is the leader.
func (n *Node) IsLeader() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.nodeID == n.leaderID
}

/*
	updates state when a heartbeat is received.

Inputs:
- leaderID: ID of the leader sending the heartbeat
*/
func (n *Node) UpdateHeartbeatFromLeader(leaderID int) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.leaderID = leaderID
	n.lastHeartbeat = time.Now()
	n.leaderAlive = true

	// Update leaderAddr from peerInfos so we know where to forward writes.
	// This handles the case where a new leader was elected while this node was down.
	for _, p := range n.peerInfos {
		if p.ID == leaderID {
			n.leaderAddr = p.Addr
			break
		}
	}
}

/*
marks the leader as not alive.
TODO: Call for an election
*/
func (n *Node) MarkLeaderDead() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.leaderAlive = false
}

/*
sets a new leader and updates related state.

Inputs:
- id: leader’s ID
- addr: leader’s address
*/
func (n *Node) SetLeader(id int, addr string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.leaderID = id
	n.leaderAddr = addr
	n.leaderAlive = true
	n.lastHeartbeat = time.Now()
}

/*
sets the local broadcaster used by this node.

Inputs:
- b: a Broadcaster implementation to notify connected WebSocket clients
*/
func (n *Node) SetBroadcaster(b Broadcaster) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.broadcaster = b
}

/*
sends a raw message to local WebSocket clients if a broadcaster is configured.
This is more like an interface tbh. The actual BroadcastRaw() is in webscoket.go
Inputs:
- msg: a byte slice containing the message to broadcast
*/
func (n *Node) BroadcastRaw(msg []byte) {
	n.mu.RLock()
	b := n.broadcaster
	n.mu.RUnlock()

	if b != nil {
		b.BroadcastRaw(msg)
	}
}
