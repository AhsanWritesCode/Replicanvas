import React, { useRef, useEffect, useState } from 'react';
import './Canvas.css';

const CANVAS_WIDTH = 100;
const CANVAS_HEIGHT = 100;
const PIXEL_SIZE = 8;

// CanvasAPI documentation was referenced extensively to learn how to draw on the canvas and manipulate pixel data
// https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API
const Canvas = ({ selectedColor, mode, brushSize, onCoordinateChange, initialCanvasState, canvasService, wsConnected }) => {
  const canvasRef = useRef(null);
  // Learned about useState from https://react.dev/reference/react/useState
  // 2D array representing pixel colors: canvasState[y][x] = '#FFFFFF'.
  const [canvasState, setCanvasState] = useState([]);
  // Track whether user is currently mouse-dragging to enable continuous drawing.
  const [isDragging, setIsDragging] = useState(false);

  // Initialize canvas with the snapshot from the server
  // Learned about useEffect from https://react.dev/reference/react/useEffect
  useEffect(() => {
    if (initialCanvasState) {
      // Unwrap the pixels array from the HTTP response object.
      const pixels = initialCanvasState.pixels || []; // Default to empty array if pixels property is missing
      setCanvasState(pixels);
    } else {
      // Fallback: initialize with white pixels if snapshot not loaded yet
      const initialState = Array(CANVAS_HEIGHT).fill(null).map(() => 
        Array(CANVAS_WIDTH).fill('#FFFFFF')
      );
      setCanvasState(initialState);
    }
  }, [initialCanvasState]);

  // Listen for WebSocket pixel updates from other clients
  useEffect(() => {
    if (!canvasService) return;

    // Subscribe: register callback to be called when server broadcasts pixel updates.
    const unsubscribe = canvasService.onPixelUpdate((update) => {
      const { x, y, color } = update;
      
      // Validate coordinates
      if (x < 0 || x >= CANVAS_WIDTH || y < 0 || y >= CANVAS_HEIGHT) {
        return;
      }

      // Update canvas state with the received pixel update
      setCanvasState(prevState => {
        const newState = prevState.map(row => [...row]);
        newState[y][x] = color;
        return newState;
      });
    });

    // Cleanup subscription on unmount
    return unsubscribe;
  }, [canvasService]);

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
        // Set fill color from state (or white as default).
        ctx.fillStyle = canvasState[y]?.[x] || '#FFFFFF';
        ctx.fillRect(x * PIXEL_SIZE, y * PIXEL_SIZE, PIXEL_SIZE, PIXEL_SIZE);
      }
    }

    // Draw grid
    ctx.strokeStyle = '#E0E0E0';
    ctx.lineWidth = 0.5;
    // Vertical grid lines.
    for (let i = 0; i <= CANVAS_WIDTH; i++) {
      ctx.beginPath();
      ctx.moveTo(i * PIXEL_SIZE, 0);
      ctx.lineTo(i * PIXEL_SIZE, CANVAS_HEIGHT * PIXEL_SIZE);
      ctx.stroke();
    }
    // Horizontal grid lines.
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
    // Calculate which logical pixel the mouse is over.
    const x = Math.floor((e.clientX - rect.left) / PIXEL_SIZE);
    const y = Math.floor((e.clientY - rect.top) / PIXEL_SIZE);
    return { x, y };
  };

  // Place pixels on canvas (local state + send to server).
  // Called on every mouse move (while dragging) or unique click.
  const placePixel = (x, y) => {
    // Bounds check: ignore clicks outside canvas.
    if (x < 0 || x >= CANVAS_WIDTH || y < 0 || y >= CANVAS_HEIGHT) return;

    const newState = canvasState.map(row => [...row]);
    // Collect all pixels to update for server broadcast.
    const pixelsToUpdate = [];

    // Pixel mode: place single pixel at cursor.
    if (mode === 'pixel') { 
      newState[y][x] = selectedColor;
      pixelsToUpdate.push({ x, y, color: selectedColor });
    } 
    // Brush mode: place NxN square of pixels centered at cursor.
    else if (mode === 'brush') {
      const radius = Math.floor(brushSize / 2);
      for (let dy = -radius; dy <= radius; dy++) {
        for (let dx = -radius; dx <= radius; dx++) {
          const nx = x + dx;
          const ny = y + dy;
          // Only draw pixels within canvas bounds.
          if (nx >= 0 && nx < CANVAS_WIDTH && ny >= 0 && ny < CANVAS_HEIGHT) {
            newState[ny][nx] = selectedColor;
            pixelsToUpdate.push({ x: nx, y: ny, color: selectedColor });
          }
        }
      }
    }

    setCanvasState(newState); // Update local canvas state (immediate visual feedback).
    
    // Broadcast pixels to server for other clients to see.
    pixelsToUpdate.forEach(pixel => {
      if (canvasService) {
        canvasService.sendPixelUpdate(pixel.x, pixel.y, pixel.color);
      }
    });
  };

  // Mouse move: update HUD cursor position, and draw if currently dragging.
  const handleMouseMove = (e) => {
    const { x, y } = getCanvasCoordinates(e);
    // Tell App.js to update HUD display (shows "X: 45  Y: 78").
    onCoordinateChange(x, y);

    // If user is holding mouse down, paint continuously.
    if (isDragging) {
      placePixel(x, y);
    }
  };

  // Mouse down: start drag, and place one pixel at click location.
  const handleMouseDown = (e) => {
    setIsDragging(true);
    const { x, y } = getCanvasCoordinates(e);
    placePixel(x, y);
  };

  // Mouse up: stop drag. No more painting until next click.
  const handleMouseUp = () => {
    setIsDragging(false);
  };

  // Mouse leave canvas: stop drag and reset HUD position.
  const handleMouseLeave = () => {
    setIsDragging(false);
    onCoordinateChange(null, null);
  };

  // Render canvas element with event handlers and dimensions.
  // Size = 800x800 pixels (100 logical x 100 logical * 8px per pixel).
  return (
    <div className="canvas-container">
      <canvas
        ref={canvasRef}
        width={CANVAS_WIDTH * PIXEL_SIZE}
        height={CANVAS_HEIGHT * PIXEL_SIZE}
        // Mouse events: coordinate tracking, drawing, drag detection.
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
