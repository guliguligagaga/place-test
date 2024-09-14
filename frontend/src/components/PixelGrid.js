import React, { useCallback, useEffect, useRef } from 'react';

const PIXEL_SIZE = 2; // Adjust this value to change the zoom level

const PixelGrid = React.memo(({ grid, onPixelClick, size, colors }) => {
    const canvasRef = useRef(null);
    const previousGridRef = useRef(null);

    const drawPixel = useCallback((ctx, x, y, colorIndex) => {
        ctx.fillStyle = colors[colorIndex];
        ctx.fillRect(x * PIXEL_SIZE, y * PIXEL_SIZE, PIXEL_SIZE, PIXEL_SIZE);
    }, [colors]);

    useEffect(() => {
        const canvas = canvasRef.current;
        if (!canvas) return;

        const ctx = canvas.getContext('2d');
        if (!ctx) return;

        const animate = () => {
            if (grid !== previousGridRef.current) {
                for (let y = 0; y < size; y++) {
                    for (let x = 0; x < size; x++) {
                        const i = y * size + x;
                        if (previousGridRef.current === null || grid[i] !== previousGridRef.current[i]) {
                            drawPixel(ctx, x, y, grid[i]);
                        }
                    }
                }
                previousGridRef.current = new Uint8Array(grid);
            }
            requestAnimationFrame(animate);
        };

        animate();

        return () => {
            cancelAnimationFrame(animate);
        };
    }, [grid, drawPixel, size]);

    const handleClick = useCallback((event) => {
        const canvas = canvasRef.current;
        if (!canvas) return;

        const rect = canvas.getBoundingClientRect();
        const x = Math.floor((event.clientX - rect.left) / PIXEL_SIZE);
        const y = Math.floor((event.clientY - rect.top) / PIXEL_SIZE);

        if (x >= 0 && x < size && y >= 0 && y < size) {
            onPixelClick(x, y);
        }
    }, [onPixelClick, size]);

    return (
        <canvas
            ref={canvasRef}
            width={size * PIXEL_SIZE}
            height={size * PIXEL_SIZE}
            onClick={handleClick}
            style={{ border: '1px solid #ddd', cursor: 'pointer' }}
        />
    );
});

export default PixelGrid;