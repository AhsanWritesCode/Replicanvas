#!/bin/bash
# Usage: ./start-replicas-demo.sh <node_number>

set -e

A_IP="10.14.135.99"   # Laptop A — runs node 1
B_IP="10.14.121.175"   # Laptop B — runs node 2
C_IP="10.14.99.58"  # Laptop C — runs node 3s
PORT=8080
# =================================================

if [ -z "$1" ]; then
    echo "Usage: $0 <node_number>   (1, 2, or 3)"
    exit 1
fi

NODE=$1
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$SCRIPT_DIR/.."
BINARY="$ROOT/bin/server"
LOG_DIR="$ROOT/logs"

mkdir -p "$ROOT/bin" "$LOG_DIR"

echo "Building server..."
(cd "$ROOT" && go build -o "$BINARY" ./cmd)
echo "Build complete."

case $NODE in
    1)
        SELF_ADDR="$A_IP:$PORT"
        PEERS="2=$B_IP:$PORT, 3=$C_IP:$PORT"
        ;;
    2)
        SELF_ADDR="$B_IP:$PORT"
        PEERS="1=$A_IP:$PORTq, 3=$C_IP:$PORT"
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

NODE_ID=$NODE \
LEADER_ID=0 \
PORT=$PORT \
SELF_ADDR="$SELF_ADDR" \
PEERS="$PEERS" \
LEADER_ADDR= \
"$BINARY" > "$LOG_DIR/node${NODE}.log" 2>&1 &
echo $! > "$LOG_DIR/node${NODE}.pid"

echo "Started Node $NODE on $SELF_ADDR  [PID $(cat "$LOG_DIR/node${NODE}.pid")]"
echo "Peers: $PEERS"
echo ""
echo "Logs: $LOG_DIR/node${NODE}.log"
echo "Run scripts/stop-replicas.sh to stop this node."
