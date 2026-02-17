import React from 'react';
import './HUD.css';

const HUD = ({ x, y }) => {
  return (
    <div className="hud">
      <div className="hud-item">
        <span className="hud-label">Coordinates:</span>
        <span className="hud-value">
          {x !== null && y !== null ? `(${x}, ${y})` : '(--, --)'}
        </span>
      </div>
    </div>
  );
};

export default HUD;
