#!/bin/bash
# Builds the server once, then starts 4 replicas on ports 8080, 8081, 8082, 8083.
# Logs go to logs/node1.log, logs/node2.log, logs/node3.log, logs/node4.log.
# Run stop-replicas.sh to kill all four.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$SCRIPT_DIR/.."
BINARY="$ROOT/bin/server"
LOG_DIR="$ROOT/logs"

mkdir -p "$ROOT/bin" "$LOG_DIR"

echo "Building server..."
(cd "$ROOT" && go build -o "$BINARY" ./cmd)
echo "Build complete."

# All peers listed with id=address format so election can map leader IDs to addresses
# Node 5 has the highest ID so it becomes initial leader (standard Bully on first boot, all snapshot timestamps are 0)

# Node 1: Follower
NODE_ID=1 \
LEADER_ID=0 \
PORT=8080 \
PEERS="2=localhost:8081,3=localhost:8082,4=localhost:8083,5=localhost:8084" \
LEADER_ADDR= \
"$BINARY" > "$LOG_DIR/node1.log" 2>&1 &
echo $! > "$LOG_DIR/node1.pid"
echo "Started Node 1 (follower) on :8080  [PID $(cat "$LOG_DIR/node1.pid")]"

# Node 2: Follower
NODE_ID=2 \
LEADER_ID=0 \
PORT=8081 \
PEERS="1=localhost:8080,3=localhost:8082,4=localhost:8083,5=localhost:8084" \
LEADER_ADDR= \
"$BINARY" > "$LOG_DIR/node2.log" 2>&1 &
echo $! > "$LOG_DIR/node2.pid"
echo "Started Node 2 (follower) on :8081  [PID $(cat "$LOG_DIR/node2.pid")]"

# Node 3: Follower
NODE_ID=3 \
LEADER_ID= \
PORT=8082 \
PEERS="1=localhost:8080,2=localhost:8081,4=localhost:8083,5=localhost:8084" \
LEADER_ADDR= \
"$BINARY" > "$LOG_DIR/node3.log" 2>&1 &
echo $! > "$LOG_DIR/node3.pid"
echo "Started Node 3 (follower) on :8082  [PID $(cat "$LOG_DIR/node3.pid")]"

# Node 4: Leader (highest ID)
NODE_ID=4 \
LEADER_ID=0 \
PORT=8083 \
PEERS="1=localhost:8080,2=localhost:8081,3=localhost:8082,5=localhost:8084" \
LEADER_ADDR= \
"$BINARY" > "$LOG_DIR/node4.log" 2>&1 &
echo $! > "$LOG_DIR/node4.pid"
echo "Started Node 4 (leader)   on :8083  [PID $(cat "$LOG_DIR/node4.pid")]"

# Node 5: Leader (highest ID)
NODE_ID=5 \
LEADER_ID=0 \
PORT=8084 \
PEERS="1=localhost:8080,2=localhost:8081,3=localhost:8082,4=localhost:8083" \
LEADER_ADDR= \
"$BINARY" > "$LOG_DIR/node5.log" 2>&1 &
echo $! > "$LOG_DIR/node5.pid"
echo "Started Node 5 (leader)   on :8084  [PID $(cat "$LOG_DIR/node4.pid")]"

echo ""
echo "All 5 replicas running."
echo "  Follower: http://localhost:8080  (Node 1)"
echo "  Follower: http://localhost:8081  (Node 2)"
echo "  Follower: http://localhost:8082  (Node 3)"
echo "  Follower: http://localhost:8083  (Node 4)"
echo "  Leader:   http://localhost:8084  (Node 5)"
echo ""
echo "Logs: $LOG_DIR/"
echo "Run scripts/stop-replicas.sh to stop all nodes."
echo "Run scripts/kill-node.sh <node_number> to kill a specific node (for fault tolerance testing)."
