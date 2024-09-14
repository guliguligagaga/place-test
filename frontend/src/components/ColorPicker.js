import React from 'react';
import { Button } from 'react-bootstrap';

const ColorPicker = React.memo(({ selectedColor, onColorSelect, colors }) => (
    <div className="mb-4">
        {colors.map((color, index) => (
            <Button
                key={index}
                variant="outline-secondary"
                style={{
                    backgroundColor: color,
                    width: '2rem',
                    height: '2rem',
                    margin: '0.25rem',
                    border: index === selectedColor ? '2px solid black' : '1px solid #ddd'
                }}
                onClick={() => onColorSelect(index)}
            />
        ))}
    </div>
));

export default ColorPicker;