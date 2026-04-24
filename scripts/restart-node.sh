#!/bin/bash
# Restarts a previously killed node.
# Usage: ./restart-node.sh <node_number>
# Example: ./restart-node.sh 5    (restarts node 5 after it was killed)
#
# The node will start with LEADER_ID pointing to itself, but it will
# receive heartbeats from the current leader within ~500ms and update
# its state automatically. If no leader exists, it will trigger an election.

if [ -z "$1" ]; then
    echo "Usage: $0 <node_number>"
    echo "Example: $0 5    (restarts node 5)"
    exit 1
fi

NODE=$1
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$SCRIPT_DIR/.."
BINARY="$ROOT/bin/server"
LOG_DIR="$ROOT/logs"

# Node config: maps node number to port and peers
case $NODE in
    1)
        PORT=8080
        PEERS="2=localhost:8081,3=localhost:8082,4=localhost:8083,5=localhost:8084"
        ;;
    2)
        PORT=8081
        PEERS="1=localhost:8080,3=localhost:8082,4=localhost:8083,5=localhost:8084"
        ;;
    3)
        PORT=8082
        PEERS="1=localhost:8080,2=localhost:8081,4=localhost:8083,5=localhost:8084"
        ;;
    4)
        PORT=8083
        PEERS="1=localhost:8080,2=localhost:8081,3=localhost:8082,5=localhost:8084"
        ;;
    5)
        PORT=8084
        PEERS="1=localhost:8080,2=localhost:8081,3=localhost:8082,4=localhost:8083"
        ;;
    *)
        echo "Unknown node number: $NODE (expected 1, 2, 3, 4, or 5)"
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
    (cd "$ROOT" && go build -o "$BINARY" ./cmd)
fi

# Start the node
# LEADER_ID is set to itself; it will discover the real leader
# via heartbeats from the current leader within ~500ms
NODE_ID=$NODE \
LEADER_ID=$NODE \
PORT=$PORT \
PEERS="$PEERS" \
LEADER_ADDR="localhost:$PORT" \
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