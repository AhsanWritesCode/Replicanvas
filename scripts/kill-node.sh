#!/bin/bash
# Kills a specific node to test fault tolerance.
# Usage: ./kill-node.sh <node_number>
# Example: ./kill-node.sh 4    (kills the leader)

if [ -z "$1" ]; then
    echo "Usage: $0 <node_number>"
    echo "Example: $0 4    (kills node 4, the initial leader)"
    exit 1
fi

NODE=$1
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="$SCRIPT_DIR/../logs"
PID_FILE="$LOG_DIR/node${NODE}.pid"

if [ ! -f "$PID_FILE" ]; then
    echo "Node $NODE is not running (no PID file found)"
    exit 1
fi

PID=$(cat "$PID_FILE")

if kill -9 "$PID" 2>/dev/null; then
    echo "Killed Node $NODE (PID $PID)"
    echo ""
    echo "What should happen now:"
    echo "  - If this was the leader, followers will detect missed heartbeats in ~1.5s"
    echo "  - An election will start automatically (Modified Bully algorithm)"
    echo "  - The node with the most recent snapshot (or highest ID if tied) wins"
    echo "  - New leader announces itself, followers update and resume"
    echo ""
    echo "Check logs to see the election:"
    echo "  tail -f $LOG_DIR/node*.log | grep election"
else
    echo "Node $NODE (PID $PID) was not running"
fi

rm -f "$PID_FILE"
