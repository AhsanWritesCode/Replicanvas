#!/bin/bash
# Stops all replica nodes started by start-replicas.sh.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="$SCRIPT_DIR/../logs"

for i in 1 2 3 4; do
    PID_FILE="$LOG_DIR/node${i}.pid"
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if kill "$PID" 2>/dev/null; then
            echo "Stopped Node $i (PID $PID)"
        else
            echo "Node $i (PID $PID) was not running"
        fi
        rm -f "$PID_FILE"
    else
        echo "No PID file for Node $i"
    fi
done

# Also kill any stray server processes that might be lingering
pkill -f "bin/server" 2>/dev/null

echo "Done."
