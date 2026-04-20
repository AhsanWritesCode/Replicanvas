#!/bin/bash
# Restarts a previously killed node.
# Usage: ./restart-node.sh <node_number>
# Example: ./restart-node.sh 4    (restarts node 4 after it was killed)
#
# The node will start with LEADER_ID pointing to itself, but it will
# receive heartbeats from the current leader within ~500ms and update
# its state automatically. If no leader exists, it will trigger an election.

if [ -z "$1" ]; then
    echo "Usage: $0 <node_number>"
    echo "Example: $0 4    (restarts node 4)"
    exit 1
fi

A_IP="10.12.119.204"   # Laptop A — runs node 1
B_IP="10.13.139.217"   # Laptop B — runs node 2
C_IP="10.13.131.22"  # Laptop C — runs node 3s
PORT=8080

NODE=$1
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$SCRIPT_DIR/.."
BINARY="$ROOT/bin/server"
LOG_DIR="$ROOT/logs"

# Node config: maps node number to port and peers
case $NODE in
    1)
        SELF_ADDR="$A_IP:$PORT"
        PEERS="2=$B_IP:$PORT, 3=$C_IP:$PORT"
        ;;
    2)
        SELF_ADDR="$B_IP:$PORT"
        PEERS="1=$A_IP:$PORT, 3=$C_IP:$PORT"
        ;;
    3)
        SELF_ADDR="$C_IP:$PORT"
        PEERS="1=$A_IP:$PORT,2=$B_IP:$PORT"
        ;;
    *)
        echo "Unknown node number: $NODE (expected 1, 2, or 3)"
        exit 1
        ;;
esac

# Check if it's already running
PID_FILE="$LOG_DIR/node${NODE}.pid"
if [ -f "$PID_FILE" ]; then
    OLD_PID=$(cat "$PID_FILE")
    if kill -0 "$OLD_PID" 2>/dev/null; then
        echo "Node $NODE is already running (PID $OLD_PID). Kill it first with kill-node.sh $NODE"
        exit 1
    fi
fi

# Build if binary doesn't exist
if [ ! -f "$BINARY" ]; then
    echo "Building server..."
    (cd "$ROOT" && go build -o "$BINARY" ./cmd/main.go)
fi

# Start the node
# LEADER_ID is set to itself, it will discover the real leader
# via heartbeats from the current leader within ~500ms
NODE_ID=$NODE \
LEADER_ID=$NODE \
PORT=$PORT \
SELF_ADDR="$SELF_ADDR" \
PEERS="$PEERS" \
LEADER_ADDR= \
"$BINARY" > "$LOG_DIR/node${NODE}.log" 2>&1 &
echo $! > "$LOG_DIR/node${NODE}.pid"

echo "Restarted Node $NODE on :$PORT  [PID $(cat "$LOG_DIR/node${NODE}.pid")]"
echo ""
echo "What happens now:"
echo "  - Node $NODE loads its snapshot from disk (if it exists)"
echo "  - Within ~500ms it will receive a heartbeat from the current leader"
echo "  - It updates its leader state and starts operating as a follower"
echo ""
echo "Check logs: tail -f $LOG_DIR/node${NODE}.log"