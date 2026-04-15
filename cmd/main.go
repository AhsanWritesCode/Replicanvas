package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/AhsanWritesCode/559-project/internal/canvas"
	"github.com/AhsanWritesCode/559-project/internal/config"
	"github.com/AhsanWritesCode/559-project/internal/node"
	"github.com/AhsanWritesCode/559-project/internal/replication"
	"github.com/AhsanWritesCode/559-project/internal/server"
	"github.com/AhsanWritesCode/559-project/internal/snapshot"
)

// entry point for the application, initializes the canvas and the HTTP and Websocket handlers and starts the server
func main() {
	c := canvas.NewCanvas(canvas.DefaultCanvasWdith, canvas.DefaultCanvasHeight)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Load snapshot from disk if it exists (recover state from previous run)
	nodeID := config.MustIntEnv("NODE_ID", 1)
	snapshotPath := fmt.Sprintf("./snapshot-node%d.json", nodeID)
	snapshotTs, err := snapshot.Load(c, snapshotPath)
	if err != nil {
		log.Printf("warning: failed to load snapshot: %v", err)
	}

	// selfAddr is this node's own address — needed for leader announcements
	// so followers know where to forward writes after an election
	selfAddr := os.Getenv("SELF_ADDR")
	if selfAddr == "" {
		// default to localhost
		selfAddr = "localhost:" + port
	}

	hbTimeout := config.MustDurationEnvMs("HB_TIMEOUT_MS", 1500)
	electionTimeout := config.MustDurationEnvMs("ELECTION_TIMEOUT_MS", 2000)

	n := node.NewNode(
		c,
		nodeID,
		config.MustIntEnv("LEADER_ID", 1),
		os.Getenv("PEERS"),
		selfAddr,
		os.Getenv("LEADER_ADDR"),
		electionTimeout,
	)

	// Set the snapshot timestamp and path so elections know how recent our data is,
	// and so SetLeader can trigger a sync from the leader when this node is a follower
	n.SetSnapshotTimestamp(snapshotTs)
	n.SetSnapshotPath(snapshotPath)

	// Save canvas to disk every 30 seconds, update snapshot timestamp after each save
	// This is to ensure that if the node crashes, it can recover its state from the snapshot
	// 30 seconds should be enough for pixel canvas, it should be negligible overhead. The worse case is we lose 30 seconds of pixel placement on a full cluster crash, which is acceptable
	snapshot.StartPeriodicSave(c, snapshotPath, 30*time.Second, func(ts int64) {
		n.SetSnapshotTimestamp(ts)
	})

	// Create the HTTP and Websocket handlers here
	httpHandler := server.NewHttpHandler(n)
	wsHandler := server.NewWSHandler(n)

	n.SetBroadcaster(wsHandler)

	// Client-facing endpoints
	http.HandleFunc("/snapshot", httpHandler.GetSnapshot)
	http.HandleFunc("/ws", wsHandler.HandleWS)

	// Internal replication endpoints (replica-to-replica)
	http.HandleFunc("/internal/forward/pixel", func(w http.ResponseWriter, r *http.Request) {
		replication.HandleForwardPixel(n, w, r)
	})
	http.HandleFunc("/internal/replicate/pixel", func(w http.ResponseWriter, r *http.Request) {
		replication.HandleReplicatePixel(n, w, r)
	})

	// Internal heartbeat and election endpoints
	http.HandleFunc("/internal/heartbeat", n.HandleHeartbeat)
	http.HandleFunc("/internal/election", n.HandleElection)
	http.HandleFunc("/internal/leader", n.HandleLeaderAnnouncement)

	// Start heartbeat loops (leader sends heartbeats, followers watch for timeout)
	hbInterval := config.MustDurationEnvMs("HB_INTERVAL_MS", 500)
	n.StartHeartbeats(hbInterval, hbTimeout)

	// Use ListenConfig with SO_REUSEADDR so we can restart quickly
	// without waiting for TIME_WAIT connections to clear on the port
	lc := net.ListenConfig{
		Control: setReuseAddr,
	}
	ln, err := lc.Listen(context.Background(), "tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", port, err)
	}

	log.Printf("Node %d running on port %s (leader=%d)", nodeID, port, config.MustIntEnv("LEADER_ID", 1))
	log.Fatal(http.Serve(ln, nil))
}
