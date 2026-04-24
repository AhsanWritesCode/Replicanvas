package canvas

//We need this package to be able to use RWLock to synchronize access to the canvas data
//More we'll be explained below
import "sync"

// Default canvas width and height dimensions (we can always change these values later depending on how big we want the canvas to be)
const (
	DefaultCanvasWdith  = 1000
	DefaultCanvasHeight = 1000
)

// This 2Dcanvas is the main data structure that will be used to store the canvas data
// It is thread safe and can be accessed by multiple goroutines
// When two clients place a pixel on the cavas at the same time, two goroutines write to Pixels[][] concurrently,
// without muxtex here, the data will run into a race conditions, Go will panic or corrupt memory
// RWMutex specifically lets multiple readers access the data (which is what the GetPixel does below and snapshot) in parallel
// While the writes to Pixels[][] (SetPixel below) get exlu exclusive access, ensuring only one goroutine can write at a time to avoid race conditions
type Canvas struct {
	mu     sync.RWMutex
	Width  int
	Height int
	Pixels [][]string
}

// Initilize the canvas with the
func NewCanvas(width, height int) *Canvas {
	//If the width or height is less than or equal to 0, return nil
	if width <= 0 || height <= 0 {
		return nil
	}

	//Create a 2D slice of strings to represent the canvas
	pixels := make([][]string, height)
	for i := range pixels {
		pixels[i] = make([]string, width)
		for j := range pixels[i] {
			pixels[i][j] = "#FFFFFF"
		}
	}
	//Return a pointer to the canvas
	return &Canvas{
		Width:  width,
		Height: height,
		Pixels: pixels,
	}
}

// Function to set a pixel on the canvas
// check if the coordinates are within the canvas boundaries set
// get the lock on the canvas to get exlusive access to the canvas data
// set the pixel color in the canvas data
// release the lock on the canvas to allow other goroutines to access the canvas data
// return true if the pixel was set successfully
func (c *Canvas) SetPixel(x, y int, color string) bool {
	// first check if the coordinates to be set are within the canvas boundaries set
	if x < 0 || x >= c.Width || y < 0 || y >= c.Height {
		return false
	}

	// get the lock on the canvas to get exlusive access to the canvas data
	c.mu.Lock()
	c.Pixels[y][x] = color
	// release the lock on the canvas to allow other goroutines to access the canvas data
	c.mu.Unlock()
	return true
}

// This function is used to get a snapshot of the pixel data on the canvas
// get the lock on the canvas to get read access to the canvas data (we are using RLock here because we are reading the data and can be accessed by multiple goroutines in parallel (meaning multiple readers can read the data at the same time))
// create a copy of the canvas data to return
// release the lock on the canvas to allow other goroutines to access the canvas data
// return the snapshot of the canvas data
func (c *Canvas) Snapshot() [][]string {
	// get the lock on the canvas to get read access to the canvas data
	c.mu.RLock()
	defer c.mu.RUnlock()
	// create a copy of the canvas data to return
	snapshot := make([][]string, c.Height)
	for i := range snapshot {
		snapshot[i] = make([]string, c.Width)
		copy(snapshot[i], c.Pixels[i])
	}
	return snapshot
}
