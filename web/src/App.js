import React, { useState, useEffect } from 'react';
import './App.css';
import Toolbar from './components/Toolbar';
import Canvas from './components/Canvas';
import HUD from './components/HUD';
import canvasService from './services/canvasService';

function App() {
  const [selectedColor, setSelectedColor] = useState('#FF0000');
  const [mode, setMode] = useState('pixel');
  const [brushSize, setBrushSize] = useState(3);
  const [cursorX, setCursorX] = useState(null);
  const [cursorY, setCursorY] = useState(null);
  // Snapshot from the HTTP endpoint used to seed the canvas state.
  const [initialCanvasState, setInitialCanvasState] = useState(null);
  // Track whether the WebSocket is connected
  const [wsConnected, setWsConnected] = useState(false);

  // Initialize WebSocket connection and fetch initial snapshot on mount
  useEffect(() => {
    const initializeConnection = async () => {
      try {
        // Fetch the initial canvas snapshot
        const snapshot = await canvasService.getSnapshot();
        setInitialCanvasState(snapshot);

        // Initialize the WebSocket connection
        await canvasService.connectWebSocket();
        setWsConnected(true);
      } catch (error) {
        console.error('Failed to initialize canvas connection:', error);
      }
    };

    initializeConnection(); // Call the async initialization function

    // Have to register the reconnect callback so the canvas re-fetches state from the new replica
    // This is only triggered when auto reconnect is active (no ?server= param)
    // Might remove this if we remove the 2 modes we have, else we'll just keep this
    canvasService.onReconnect(async () => {
      try {
        const snapshot = await canvasService.getSnapshot();
        setInitialCanvasState(snapshot);
        setWsConnected(true);
        console.log('Reconnected and refreshed canvas from new replica');
      } catch (err) {
        console.error('Failed to refresh canvas after reconnect:', err);
      }
    });

    // Cleanup on unmount
    return () => {
      canvasService.disconnect();
    };
  }, []);

  // Update HUD cursor position as the mouse moves over the canvas.
  const handleCoordinateChange = (x, y) => {
    setCursorX(x);
    setCursorY(y);
  };

  return (
    <div className="App">
      <header className="app-header">
        <h1>Replicanvas</h1>
      </header>
      
      <Toolbar
        selectedColor={selectedColor}
        onColorChange={setSelectedColor}
        mode={mode}
        onModeChange={setMode}
        brushSize={brushSize}
        onBrushSizeChange={setBrushSize}
      />
      
      <Canvas
        selectedColor={selectedColor}
        mode={mode}
        brushSize={brushSize}
        onCoordinateChange={handleCoordinateChange}
        initialCanvasState={initialCanvasState}
        canvasService={canvasService}
        wsConnected={wsConnected}
      />
      
      <HUD x={cursorX} y={cursorY} />
    </div>
  );
}

export default App;
