import React from 'react';
import { Button } from 'react-bootstrap';

export const COLORS = ['#FF0000', '#00FF00', '#0000FF', '#FFFF00', '#FF00FF', '#00FFFF', '#FFFFFF', '#000000'];

const ColorPicker = React.memo(({ selectedColor, onColorSelect }) => (
    <div className="mb-4">
        {COLORS.map(color => (
            <Button
                key={color}
                variant="outline-secondary"
                style={{
                    backgroundColor: color,
                    width: '2rem',
                    height: '2rem',
                    margin: '0.25rem',
                    border: color === selectedColor ? '2px solid black' : '1px solid #ddd'
                }}
                onClick={() => onColorSelect(color)}
            />
        ))}
    </div>
));

export default ColorPicker;