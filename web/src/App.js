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
        // pick the least-loaded server
        const replica = await canvasService.selectBestReplica();

        // fetch the initial canvas snapshot from that server
        const snapshot = await canvasService.getSnapshot(replica.http);
        setInitialCanvasState(snapshot);

        // connect to its WebSocket
        await canvasService.connectWebSocket(replica.ws);
        setWsConnected(true);
      } catch (error) {
        console.error('Failed to initialize canvas connection:', error);
      }
    };

    initializeConnection(); // Call the async initialization function

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
