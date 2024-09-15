import React, { useCallback, useEffect, useRef, useMemo } from 'react';
import PropTypes from 'prop-types';

const PIXEL_SIZE = 2; // Adjust this value to change the zoom level

const PixelGrid = React.memo(({ grid, onPixelClick, size, colors, quadrants, onVisibilityChange }) => {
    const canvasRef = useRef(null);
    const previousGridRef = useRef(null);

    const drawPixel = useCallback((ctx, x, y, colorIndex) => {
        ctx.fillStyle = colors[colorIndex];
        ctx.fillRect(x * PIXEL_SIZE, y * PIXEL_SIZE, PIXEL_SIZE, PIXEL_SIZE);
    }, [colors]);

    const visibleQuadrants = useMemo(() => {
        // Calculate visible quadrants based on current view
        // This is a placeholder implementation
        return new Set(quadrants.map(q => q.id));
    }, [quadrants]);

    useEffect(() => {
        onVisibilityChange(Array.from(visibleQuadrants));
    }, [visibleQuadrants, onVisibilityChange]);

    useEffect(() => {
        const canvas = canvasRef.current;
        if (!canvas) return;

        const ctx = canvas.getContext('2d');
        if (!ctx) return;

        let animationFrameId;

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
            animationFrameId = requestAnimationFrame(animate);
        };

        animate();

        return () => {
            cancelAnimationFrame(animationFrameId);
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

PixelGrid.propTypes = {
    grid: PropTypes.instanceOf(Uint8Array).isRequired,
    onPixelClick: PropTypes.func.isRequired,
    size: PropTypes.number.isRequired,
    colors: PropTypes.arrayOf(PropTypes.string).isRequired,
    quadrants: PropTypes.arrayOf(PropTypes.shape({
        id: PropTypes.number.isRequired,
        x: PropTypes.number.isRequired,
        y: PropTypes.number.isRequired,
    })).isRequired,
    onVisibilityChange: PropTypes.func.isRequired,
};

export default PixelGrid;