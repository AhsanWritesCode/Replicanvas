import React, { useRef, useEffect, useState } from 'react';
import './Canvas.css';

const CANVAS_WIDTH = 100;
const CANVAS_HEIGHT = 100;
const PIXEL_SIZE = 8;

const Canvas = ({ selectedColor, mode, brushSize, onCoordinateChange }) => {
  const canvasRef = useRef(null);
  const [canvasState, setCanvasState] = useState([]);
  const [isDragging, setIsDragging] = useState(false);

  // Initialize canvas with white pixels
  useEffect(() => {
    const initialState = Array(CANVAS_HEIGHT).fill(null).map(() => 
      Array(CANVAS_WIDTH).fill('#FFFFFF')
    );
    setCanvasState(initialState);
  }, []);

  // Draw the canvas
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    
    // Clear canvas
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    // Draw pixels
    for (let y = 0; y < CANVAS_HEIGHT; y++) {
      for (let x = 0; x < CANVAS_WIDTH; x++) {
        ctx.fillStyle = canvasState[y]?.[x] || '#FFFFFF';
        ctx.fillRect(x * PIXEL_SIZE, y * PIXEL_SIZE, PIXEL_SIZE, PIXEL_SIZE);
      }
    }

    // Draw grid
    ctx.strokeStyle = '#E0E0E0';
    ctx.lineWidth = 0.5;
    for (let i = 0; i <= CANVAS_WIDTH; i++) {
      ctx.beginPath();
      ctx.moveTo(i * PIXEL_SIZE, 0);
      ctx.lineTo(i * PIXEL_SIZE, CANVAS_HEIGHT * PIXEL_SIZE);
      ctx.stroke();
    }
    for (let i = 0; i <= CANVAS_HEIGHT; i++) {
      ctx.beginPath();
      ctx.moveTo(0, i * PIXEL_SIZE);
      ctx.lineTo(CANVAS_WIDTH * PIXEL_SIZE, i * PIXEL_SIZE);
      ctx.stroke();
    }
  }, [canvasState]);

  const getCanvasCoordinates = (e) => {
    const canvas = canvasRef.current;
    const rect = canvas.getBoundingClientRect();
    const x = Math.floor((e.clientX - rect.left) / PIXEL_SIZE);
    const y = Math.floor((e.clientY - rect.top) / PIXEL_SIZE);
    return { x, y };
  };

  const placePixel = (x, y) => {
    if (x < 0 || x >= CANVAS_WIDTH || y < 0 || y >= CANVAS_HEIGHT) return;

    const newState = canvasState.map(row => [...row]);

    if (mode === 'pixel') {
      // Place single pixel
      newState[y][x] = selectedColor;
    } else if (mode === 'brush') {
      // Place multiple pixels in a brush pattern
      const radius = Math.floor(brushSize / 2);
      for (let dy = -radius; dy <= radius; dy++) {
        for (let dx = -radius; dx <= radius; dx++) {
          const nx = x + dx;
          const ny = y + dy;
          if (nx >= 0 && nx < CANVAS_WIDTH && ny >= 0 && ny < CANVAS_HEIGHT) {
            // Simple square brush
            newState[ny][nx] = selectedColor;
          }
        }
      }
    }

    setCanvasState(newState);
  };

  const handleMouseMove = (e) => {
    const { x, y } = getCanvasCoordinates(e);
    onCoordinateChange(x, y);

    if (isDragging) {
      placePixel(x, y);
    }
  };

  const handleMouseDown = (e) => {
    setIsDragging(true);
    const { x, y } = getCanvasCoordinates(e);
    placePixel(x, y);
  };

  const handleMouseUp = () => {
    setIsDragging(false);
  };

  const handleMouseLeave = () => {
    setIsDragging(false);
    onCoordinateChange(null, null);
  };

  return (
    <div className="canvas-container">
      <canvas
        ref={canvasRef}
        width={CANVAS_WIDTH * PIXEL_SIZE}
        height={CANVAS_HEIGHT * PIXEL_SIZE}
        onMouseMove={handleMouseMove}
        onMouseDown={handleMouseDown}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseLeave}
        className="pixel-canvas"
      />
    </div>
  );
};

export default Canvas;
