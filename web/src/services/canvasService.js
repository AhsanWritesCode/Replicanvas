// build the replicas list from env var, default to just localhost:8080
// follows the same REACT_APP_ env var pattern as the rest of the codebase
// e.g. REACT_APP_REPLICAS=http://localhost:8080,http://localhost:8081,http://localhost:8082
// Learned from https://developer.mozilla.org/en-US/docs/Web/API/WebSocket
const replicaUrls = (process.env.REACT_APP_REPLICAS || 'http://localhost:8080')
  .split(',')
  .map(u => u.trim());

const REPLICAS = replicaUrls.map(http => ({
  http,
  ws: http.replace('http', 'ws') + '/ws'
}));

// CanvasService class to manage communication with the backend
// Learnt about classes from https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Statements/class
class CanvasService {
  constructor() {
    this.ws = null; // WebSocket instance
    this.messageHandlers = []; // Handlers to call when a WebSocket message is received
  }

  // picks the replica with the fewest active connections
  // if theres a tie, pick randomly from the tied ones
  // learned about Promise.allSettled from https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise/allSettled
  async selectBestReplica() {
    // fire GET /status on all replicas in parallel
    const results = await Promise.allSettled(
      REPLICAS.map(async (replica) => {
        const response = await fetch(`${replica.http}/status`);
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();
        return { replica, connections: data.connections };
      })
    );

    // filter out replicas that didn't respond
    const alive = results
      .filter(r => r.status === 'fulfilled')
      .map(r => r.value);

    if (alive.length === 0) {
      throw new Error('All replicas are unreachable');
    }

    // find the lowest connection count
    const minCount = Math.min(...alive.map(r => r.connections));

    // grab all replicas tied at the minimum
    const tied = alive.filter(r => r.connections === minCount);

    // random tiebreaker
    const chosen = tied[Math.floor(Math.random() * tied.length)];

    console.log(`Selected replica ${chosen.replica.http} (${chosen.connections} connections)`);
    return chosen.replica;
  }

  // Fetch the initial canvas snapshot from the HTTP endpoint
  // Learned about async/await and fetch API from https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Statements/async_function 
  // and https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch
  async getSnapshot(baseUrl) {
    try {
      const response = await fetch(`${baseUrl}/snapshot`); // Fetch API 
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
  connectWebSocket(wsUrl) {
    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(wsUrl); // connect to the selected server

        // If the connection is successful, log it and resolve the promise
        this.ws.onopen = () => {
          console.log('WebSocket connected to', wsUrl);
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
          console.log('WebSocket disconnected');
        };
      } catch (error) {
        console.error('Error creating WebSocket:', error);
        reject(error);
      }
    });
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
