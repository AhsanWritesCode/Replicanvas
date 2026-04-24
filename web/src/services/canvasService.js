// Server spins up on localhost:8080, and the WebSocket endpoint is at ws://localhost:8080/ws
// Learned from https://developer.mozilla.org/en-US/docs/Web/API/WebSocket

// All known replicas in the cluster
const ALL_REPLICAS = ["10.13.143.207:8080", "10.13.97.171:8080", "10.13.131.115:8080"]; // Change to IPs of machines

// Two modes:
// 1. With ?server= param: connects to that specific replica only, no auto reconnect.
//    If it dies, you're stuck. Good for demo4 to show failure.
//    Example: http://localhost:3000/?server=localhost:8083
//
// 2. Without ?server= param: auto reconnects to the next available replica if the
//    current one dies. Good for demo4 to show fault tolerance.
//    Example: http://localhost:3000/
const params = new URLSearchParams(window.location.search);
const fixedServer = params.get("server");

// CanvasService class to manage communication with the backend
// Learnt about classes from https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Statements/class
class CanvasService {
  constructor() {
    this.ws = null; // WebSocket instance
    this.messageHandlers = []; // Handlers to call when a WebSocket message is received
    this.currentServer = fixedServer || ALL_REPLICAS[Math.floor(Math.random() * ALL_REPLICAS.length)]; // Which replica we're connected to
    this.autoReconnect = !fixedServer; // Only auto reconnect if no fixed server was specified
    this.reconnecting = false; // Prevents multiple reconnect attempts at the same time
    this.snapshotCallback = null; // Callback to refresh the canvas after reconnecting
  }

  // Set a callback that will be called when we reconnect to a new replica
  // so the canvas can re-fetch the snapshot from the new server
  onReconnect(callback) {
    this.snapshotCallback = callback;
  }

  // Fetch the initial canvas snapshot from the HTTP endpoint
  // Learned about async/await and fetch API from https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Statements/async_function
  // and https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch
  async getSnapshot() {
    try {
      const response = await fetch(`http://${this.currentServer}/snapshot`);
      // Check if the response is OK (status in the range 200-299)
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json(); // Parse the JSON response
      return data; // Return the snapshot data
    } catch (error) {
      console.error('Error fetching snapshot:', error);
      throw error;
    }
  }

  // Initialize WebSocket connection for real-time updates
  connectWebSocket() {
    return new Promise((resolve, reject) => {
      try {
        const wsUrl = `ws://${this.currentServer}/ws`;
        this.ws = new WebSocket(wsUrl);

        // If the connection is successful, log it and resolve the promise
        this.ws.onopen = () => {
          console.log(`WebSocket connected to ${this.currentServer}`);
          resolve();
        };

        // Listen for messages from the server and notify registered handlers
        this.ws.onmessage = (event) => {
          try {
            const data = JSON.parse(event.data);
            // Notify all registered handlers of the update
            this.messageHandlers.forEach(handler => handler(data));
          } catch (error) {
            console.error('Error parsing WebSocket message:', error);
          }
        };

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error);
          reject(error);
        };

        this.ws.onclose = () => {
          console.log(`WebSocket disconnected from ${this.currentServer}`);
          // If auto reconnect is enabled (no ?server= param), try to connect to another replica
          if (this.autoReconnect) {
            this.tryReconnect();
          }
        };
      } catch (error) {
        console.error('Error creating WebSocket:', error);
        reject(error);
      }
    });
  }

  // Try to reconnect to any available replica in the cluster.
  // Cycles through all replicas, skipping the one that just died,
  // until it finds one that responds.
  async tryReconnect() {
    if (this.reconnecting) return;
    this.reconnecting = true;

    console.log('Attempting to reconnect to another replica...');

    const deadServer = this.currentServer;

    for (const replica of ALL_REPLICAS) {
      // Skip the one that just died
      if (replica === deadServer) continue;

      try {
        // Check if this replica is alive by fetching its snapshot
        const response = await fetch(`http://${replica}/snapshot`, {
          signal: AbortSignal.timeout(2000)
        });
        if (!response.ok) continue;

        // This replica is alive, switch to it
        console.log(`Reconnecting to ${replica}`);
        this.currentServer = replica;

        // Connect WebSocket to the new replica
        await this.connectWebSocket();

        // Re-fetch the snapshot so the canvas updates to the new replica's state
        if (this.snapshotCallback) {
          this.snapshotCallback();
        }

        this.reconnecting = false;
        return;
      } catch (err) {
        // This replica is also down, try the next one
        console.log(`Replica ${replica} is unreachable, trying next...`);
        continue;
      }
    }

    console.error('All replicas are unreachable. Could not reconnect.');
    this.reconnecting = false;
  }

  // Register a handler to receive WebSocket messages
  onPixelUpdate(handler) {
    this.messageHandlers.push(handler);
    // Return an unsubscribe function
    return () => {
      this.messageHandlers = this.messageHandlers.filter(h => h !== handler);
    };
  }

  // Send a pixel update through the WebSocket
  sendPixelUpdate(x, y, color) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      const update = {
        x,
        y,
        color
      };
      this.ws.send(JSON.stringify(update));
    } else {
      console.warn('WebSocket is not connected');
    }
  }

  // Disconnect WebSocket
  disconnect() {
    // Disable auto reconnect so closing doesn't trigger tryReconnect
    this.autoReconnect = false;
    if (this.ws) {
      this.ws.close(); // Close the WebSocket connection
      this.ws = null; // Clear the WebSocket reference
    }
  }
}

// Export a singleton instance
// This allows us to use the same CanvasService instance across the entire app, maintaining a shared state.
// Learned about singleton pattern from https://refactoring.guru/design-patterns/singleton
export default new CanvasService();
