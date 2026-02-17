import React from 'react';
import './Toolbar.css';

const COLORS = [
  '#FF0000',
  '#FF8800',
  '#FFFF00',
  '#00FF00',
  '#00FFFF',
  '#0000FF',
  '#8800FF',
  '#FF00FF',
  '#FFFFFF',
  '#000000',
  '#888888',
  '#FFC0CB',
];

const Toolbar = ({ selectedColor, onColorChange, mode, onModeChange, brushSize, onBrushSizeChange }) => {
  return (
    <div className="toolbar">
      <div className="toolbar-section">
        <h3>Mode</h3>
        <div className="mode-selector">
          <button
            className={`mode-button ${mode === 'pixel' ? 'active' : ''}`}
            onClick={() => onModeChange('pixel')}
          >
            Single Pixel
          </button>
          <button
            className={`mode-button ${mode === 'brush' ? 'active' : ''}`}
            onClick={() => onModeChange('brush')}
          >
            Brush
          </button>
        </div>
        {mode === 'brush' && (
          <div className="brush-size-control">
            <label>Brush Size: {brushSize}x{brushSize}</label>
            <input
              type="range"
              min="1"
              max="10"
              value={brushSize}
              onChange={(e) => onBrushSizeChange(parseInt(e.target.value))}
            />
          </div>
        )}
      </div>

      <div className="toolbar-section">
        <h3>Color</h3>
        <div className="color-palette">
          {COLORS.map((color) => (
            <button
              key={color}
              className={`color-button ${selectedColor === color ? 'selected' : ''}`}
              style={{ backgroundColor: color }}
              onClick={() => onColorChange(color)}
              title={color}
            />
          ))}
        </div>
        <div className="current-color-display">
          <span>Selected: </span>
          <div 
            className="current-color-box" 
            style={{ backgroundColor: selectedColor }}
          />
          <span>{selectedColor}</span>
        </div>
      </div>
    </div>
  );
};

export default Toolbar;
