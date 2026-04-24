package models

// PixelUpdate is the message format for pixel updates (client WS + replica forwarding/replication)
type PixelUpdate struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Color string `json:"color"`
}
