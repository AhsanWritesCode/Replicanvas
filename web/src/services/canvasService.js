// Server spins up on localhost:8080, and the WebSocket endpoint is at ws://localhost:8080/ws
// Learned from https://developer.mozilla.org/en-US/docs/Web/API/WebSocket
const API_BASE_URL = 'http://localhost:8080';
const WS_URL = 'ws://localhost:8080/ws';

// CanvasService class to manage communication with the backend
// Learnt about classes from https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Statements/class
class CanvasService {
  constructor() {
    this.ws = null; // WebSocket instance
    this.messageHandlers = []; // Handlers to call when a WebSocket message is received
  }

  // Fetch the initial canvas snapshot from the HTTP endpoint
  // Learned about async/await and fetch API from https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Statements/async_function 
  // and https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch
  async getSnapshot() {
    try {
      const response = await fetch(`${API_BASE_URL}/snapshot`); // Fetch API 
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
        this.ws = new WebSocket(WS_URL); // Create a new WebSocket connection to the server

        // If the connection is successful, log it and resolve the promise
        this.ws.onopen = () => {
          console.log('WebSocket connected');
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
export default new CanvasService();
