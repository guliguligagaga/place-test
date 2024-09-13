import React, { useCallback, useEffect, useRef } from 'react';

const GRID_SIZE = 100;
const PIXEL_SIZE = 5;

const PixelGrid = React.memo(({ grid, onPixelClick }) => {
    const canvasRef = useRef(null);

    useEffect(() => {
        const canvas = canvasRef.current;
        if (!canvas) return;

        const ctx = canvas.getContext('2d');
        if (!ctx) return;

        grid.forEach((color, index) => {
            const x = (index % GRID_SIZE) * PIXEL_SIZE;
            const y = Math.floor(index / GRID_SIZE) * PIXEL_SIZE;
            ctx.fillStyle = color;
            ctx.fillRect(x, y, PIXEL_SIZE, PIXEL_SIZE);
        });
    }, [grid]);

    const handleClick = useCallback((event) => {
        const canvas = canvasRef.current;
        if (!canvas) return;

        const rect = canvas.getBoundingClientRect();
        const x = Math.floor((event.clientX - rect.left) / PIXEL_SIZE);
        const y = Math.floor((event.clientY - rect.top) / PIXEL_SIZE);
        const index = y * GRID_SIZE + x;
        onPixelClick(index);
    }, [onPixelClick]);

    return (
        <canvas
            ref={canvasRef}
            width={GRID_SIZE * PIXEL_SIZE}
            height={GRID_SIZE * PIXEL_SIZE}
            onClick={handleClick}
            style={{ border: '1px solid #ddd', cursor: 'pointer' }}
        />
    );
});

export default PixelGrid;