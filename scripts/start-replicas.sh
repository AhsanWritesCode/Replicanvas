#!/bin/bash
# Builds the server once, then starts 3 replicas on ports 8080, 8081, 8082.
# Logs go to logs/node1.log, logs/node2.log, logs/node3.log.
# Run stop-replicas.sh to kill all three.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$SCRIPT_DIR/.."
BINARY="$ROOT/bin/server"
LOG_DIR="$ROOT/logs"

mkdir -p "$ROOT/bin" "$LOG_DIR"

echo "Building server..."
(cd "$ROOT" && go build -o "$BINARY" ./cmd/main.go)
echo "Build complete."

# Node 1 — Leader
NODE_ID=1 \
LEADER_ID=1 \
PORT=8080 \
PEERS="localhost:8081,localhost:8082" \
LEADER_ADDR="localhost:8080" \
"$BINARY" > "$LOG_DIR/node1.log" 2>&1 &
echo $! > "$LOG_DIR/node1.pid"
echo "Started Node 1 (leader) on :8080  [PID $(cat "$LOG_DIR/node1.pid")]"

# Node 2 — Follower
NODE_ID=2 \
LEADER_ID=1 \
PORT=8081 \
PEERS="localhost:8080,localhost:8082" \
LEADER_ADDR="localhost:8080" \
"$BINARY" > "$LOG_DIR/node2.log" 2>&1 &
echo $! > "$LOG_DIR/node2.pid"
echo "Started Node 2 (follower) on :8081  [PID $(cat "$LOG_DIR/node2.pid")]"

# Node 3 — Follower
NODE_ID=3 \
LEADER_ID=1 \
PORT=8082 \
PEERS="localhost:8080,localhost:8081" \
LEADER_ADDR="localhost:8080" \
"$BINARY" > "$LOG_DIR/node3.log" 2>&1 &
echo $! > "$LOG_DIR/node3.pid"
echo "Started Node 3 (follower) on :8082  [PID $(cat "$LOG_DIR/node3.pid")]"

echo ""
echo "All 3 replicas running."
echo "  Leader:   http://localhost:8080"
echo "  Follower: http://localhost:8081"
echo "  Follower: http://localhost:8082"
echo ""
echo "Logs: $LOG_DIR/"
echo "Run scripts/stop-replicas.sh to stop all nodes."