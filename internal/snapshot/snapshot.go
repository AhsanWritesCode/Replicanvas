package snapshot

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/AhsanWritesCode/559-project/internal/canvas"
)

/*
SnapshotData is the structure saved to disk.
Contains the full canvas state and a timestamp of when the snapshot was taken.

Fields:
  - Width: canvas width
  - Height: canvas height
  - Pixels: the full 2D pixel array
  - Timestamp: Unix timestamp (seconds) of when this snapshot was created,
    used during leader elections to prefer the node with the most recent data
*/
type SnapshotData struct {
	Width     int        `json:"width"`
	Height    int        `json:"height"`
	Pixels    [][]string `json:"pixels"`
	Timestamp int64      `json:"timestamp"`
}

/*
Save writes the current canvas state to a JSON file on disk.

Inputs:
- c: the canvas to snapshot
- path: file path to write the snapshot to (e.g. "./snapshot.json")

Returns:
- int64: the Unix timestamp of the snapshot (to pass to node.SetSnapshotTimestamp)
- error: if the file write fails
*/
func Save(c *canvas.Canvas, path string) (int64, error) {
	ts := time.Now().Unix()

	data := SnapshotData{
		Width:     c.Width,
		Height:    c.Height,
		Pixels:    c.Snapshot(),
		Timestamp: ts,
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return 0, err
	}

	if err := os.WriteFile(path, bytes, 0644); err != nil {
		return 0, err
	}

	return ts, nil
}

/*
Load reads a snapshot from disk and applies it to the canvas.
Called on startup to recover state from a previous run

Inputs:
- c: the canvas to restore state into
- path: file path to read the snapshot from

Returns:
  - int64: the timestamp from the snapshot (to pass to node.SetSnapshotTimestamp)
  - error: if the file doesn't exist or can't be parsed.
    If the file doesn't exist, returns 0 with no error (fresh start, no snapshot to load).
*/
func Load(c *canvas.Canvas, path string) (int64, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No snapshot file means fresh start, not an error
			return 0, nil
		}
		return 0, err
	}

	var data SnapshotData
	if err := json.Unmarshal(bytes, &data); err != nil {
		return 0, err
	}

	// Apply the snapshot pixels to the canvas
	for y := 0; y < data.Height && y < c.Height; y++ {
		for x := 0; x < data.Width && x < c.Width; x++ {
			c.SetPixel(x, y, data.Pixels[y][x])
		}
	}

	log.Printf("[snapshot] loaded canvas from %s (timestamp=%d)", path, data.Timestamp)
	return data.Timestamp, nil
}

/*
StartPeriodicSave runs a background goroutine that saves the canvas to disk
at a regular interval (30 seconds for our case, we'll test and see how it performs)

Inputs:
  - c: the canvas to snapshot
  - path: file path to write snapshots to
  - interval: how often to save
  - onSave: callback called after each successful save with the snapshot timestamp,
    used to update the node's snapshot timestamp for elections
*/
func StartPeriodicSave(c *canvas.Canvas, path string, interval time.Duration, onSave func(int64)) {
	go func() {
		// we'll use a ticker to save the canvas to disk at the regular interval
		t := time.NewTicker(interval)
		defer t.Stop()

		for range t.C {
			ts, err := Save(c, path)
			if err != nil {
				log.Printf("[snapshot] save failed: %v", err)
				continue
			}
			log.Printf("[snapshot] saved canvas to %s (timestamp=%d)", path, ts)
			if onSave != nil {
				// call the callback function to update the node's snapshot timestamp for elections
				onSave(ts)
			}
		}
	}()
}
