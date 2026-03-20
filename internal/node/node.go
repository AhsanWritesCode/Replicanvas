package node

import (
	"net/http"
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
- LastHeartbeat: last time a heartbeat was received
- LeaderAlive: whether the leader is alive
- HTTPClient: client used to send requests to peers
*/
type Node struct {
	Canvas *canvas.Canvas

	mu sync.RWMutex

	nodeID   int
	leaderID int
	peers    []string

	leaderAddr    string
	lastHeartbeat time.Time
	leaderAlive   bool

	broadcaster Broadcaster

	httpClient *http.Client
}

/*
NewNode creates and initializes a new Node.

Inputs:
- c: shared canvas state
- nodeID: this node’s ID
- leaderID: current leader’s ID
- peersCSV: comma-separated list of peer addresses
- leaderAddr: address of the leader
*/
func NewNode(c *canvas.Canvas, nodeID, leaderID int, peersCSV, leaderAddr string) *Node {
	peers := []string{}
	for _, p := range strings.Split(peersCSV, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			peers = append(peers, p)
		}
	}

	return &Node{
		Canvas: c,

		nodeID:   nodeID,
		leaderID: leaderID,
		peers:    peers,

		leaderAddr:    leaderAddr,
		lastHeartbeat: time.Now(),
		leaderAlive:   true,

		httpClient: &http.Client{
			Timeout: 800 * time.Millisecond,
		},
	}
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
