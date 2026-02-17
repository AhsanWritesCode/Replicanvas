import React, { useState } from 'react';
import './App.css';
import Toolbar from './components/Toolbar';
import Canvas from './components/Canvas';
import HUD from './components/HUD';

function App() {
  const [selectedColor, setSelectedColor] = useState('#FF0000');
  const [mode, setMode] = useState('pixel');
  const [brushSize, setBrushSize] = useState(3);
  const [cursorX, setCursorX] = useState(null);
  const [cursorY, setCursorY] = useState(null);

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
      />
      
      <HUD x={cursorX} y={cursorY} />
    </div>
  );
}

export default App;
