import React, { useRef, useState, useEffect, useCallback } from 'react';
import PropTypes from 'prop-types';

const INITIAL_ZOOM = 1;
const MAX_ZOOM = 40;
const MIN_ZOOM = 1;

const PixelGrid = React.memo(({ grid, onPixelClick, size, colors, connectedClients }) => {
    const canvasRef = useRef(null);
    const [zoom, setZoom] = useState(INITIAL_ZOOM);
    const [offset, setOffset] = useState({ x: 0, y: 0 });
    const [isDragging, setIsDragging] = useState(false);
    const [hoveredPixel, setHoveredPixel] = useState(null);
    const lastMousePosRef = useRef({ x: 0, y: 0 });

    const drawGrid = useCallback(() => {
        const canvas = canvasRef.current;
        if (!canvas) return;
        const ctx = canvas.getContext('2d');
        const canvasSize = Math.min(canvas.width, canvas.height);
        const scaleFactor = canvasSize / size;

        ctx.clearRect(0, 0, canvas.width, canvas.height);

        ctx.save();
        ctx.scale(zoom, zoom);
        ctx.translate(-offset.x, -offset.y);

        // Draw pixels
        for (let y = 0; y < size; y++) {
            for (let x = 0; x < size; x++) {
                const index = y * size + x;
                const colorIndex = grid[index];
                ctx.fillStyle = colors[colorIndex];
                ctx.fillRect(x * scaleFactor, y * scaleFactor, scaleFactor, scaleFactor);
            }
        }

        // Draw grid lines
        ctx.strokeStyle = 'rgba(200, 200, 200, 0.5)';
        ctx.lineWidth = 0.5 / zoom;
        for (let x = 0; x <= size; x++) {
            ctx.beginPath();
            ctx.moveTo(x * scaleFactor, 0);
            ctx.lineTo(x * scaleFactor, size * scaleFactor);
            ctx.stroke();
        }
        for (let y = 0; y <= size; y++) {
            ctx.beginPath();
            ctx.moveTo(0, y * scaleFactor);
            ctx.lineTo(size * scaleFactor, y * scaleFactor);
            ctx.stroke();
        }

        ctx.restore();

        // Highlight hovered pixel
        if (hoveredPixel) {
            const { x, y } = hoveredPixel;
            ctx.strokeStyle = 'rgba(255, 255, 255, 0.8)';
            ctx.lineWidth = 2 / zoom;
            ctx.strokeRect(
                (x - offset.x) * scaleFactor * zoom,
                (y - offset.y) * scaleFactor * zoom,
                scaleFactor * zoom,
                scaleFactor * zoom
            );
        }
    }, [grid, size, colors, zoom, offset, hoveredPixel, connectedClients]);

    useEffect(() => {
        drawGrid();
    }, [drawGrid]);

    useEffect(() => {
        const canvas = canvasRef.current;
        if (!canvas) return;
        const updateCanvasSize = () => {
            const containerWidth = canvas.parentElement.clientWidth;
            const containerHeight = canvas.parentElement.clientHeight;
            const canvasSize = Math.min(containerWidth, containerHeight);
            canvas.width = canvasSize;
            canvas.height = canvasSize;
            drawGrid();
        };

        updateCanvasSize();
        window.addEventListener('resize', updateCanvasSize);

        return () => {
            window.removeEventListener('resize', updateCanvasSize);
        };
    }, [drawGrid]);

    const handleWheel = useCallback((event) => {
        event.preventDefault();
        const zoomFactor = event.deltaY > 0 ? 0.9 : 1.1;
        setZoom((prevZoom) => {
            const newZoom = Math.max(MIN_ZOOM, Math.min(MAX_ZOOM, prevZoom * zoomFactor));
            const canvasRect = canvasRef.current.getBoundingClientRect();
            const mouseX = event.clientX - canvasRect.left;
            const mouseY = event.clientY - canvasRect.top;

            setOffset((prevOffset) => {
                const newOffsetX = mouseX / prevZoom + prevOffset.x - mouseX / newZoom;
                const newOffsetY = mouseY / prevZoom + prevOffset.y - mouseY / newZoom;
                
                // Calculate the maximum allowed offset
                const maxOffsetX = size - size / newZoom;
                const maxOffsetY = size - size / newZoom;
                
                return {
                    x: Math.max(0, Math.min(newOffsetX, maxOffsetX)),
                    y: Math.max(0, Math.min(newOffsetY, maxOffsetY))
                };
            });

            return newZoom;
        });
    }, [size]);

    const handleMouseDown = useCallback((event) => {
        setIsDragging(true);
        lastMousePosRef.current = { x: event.clientX, y: event.clientY };
    }, []);

     const handleMouseMove = useCallback((event) => {
         if (isDragging) {
             const dx = event.clientX - lastMousePosRef.current.x;
             const dy = event.clientY - lastMousePosRef.current.y;

             setOffset((prevOffset) => {
                 const newOffsetX = prevOffset.x - dx / zoom;
                 const newOffsetY = prevOffset.y - dy / zoom;

                 // Calculate the maximum allowed offset
                 const maxOffsetX = size - size / zoom;
                 const maxOffsetY = size - size / zoom;

                 // Clamp the offset values
                 return {
                     x: Math.max(0, Math.min(newOffsetX, maxOffsetX)),
                     y: Math.max(0, Math.min(newOffsetY, maxOffsetY))
                 };
             });

             lastMousePosRef.current = { x: event.clientX, y: event.clientY };
         } else {
             const canvas = canvasRef.current;
             if (!canvas) return;

             const rect = canvas.getBoundingClientRect();
             const canvasSize = rect.width;
             const scaleFactor = canvasSize / size;
             const x = Math.floor((event.clientX - rect.left) / (scaleFactor * zoom) + offset.x);
             const y = Math.floor((event.clientY - rect.top) / (scaleFactor * zoom) + offset.y);

             if (x >= 0 && x < size && y >= 0 && y < size) {
                 setHoveredPixel({ x, y });
             } else {
                 setHoveredPixel(null);
             }
         }
     }, [isDragging, zoom, size, offset]);

    const handleMouseUp = useCallback(() => {
        setIsDragging(false);
    }, []);

    const handleClick = useCallback((event) => {
        const canvas = canvasRef.current;
        if (!canvas) return;
        const rect = canvas.getBoundingClientRect();
        const canvasSize = rect.width;
        const scaleFactor = canvasSize / size;
        const x = Math.floor((event.clientX - rect.left) / (scaleFactor * zoom) + offset.x);
        const y = Math.floor((event.clientY - rect.top) / (scaleFactor * zoom) + offset.y);

        if (x >= 0 && x < size && y >= 0 && y < size) {
            onPixelClick(x, y);
        }
    }, [onPixelClick, size, zoom, offset]);

    useEffect(() => {
        const canvas = canvasRef.current;
        if (!canvas) return;
        canvas.addEventListener('wheel', handleWheel, { passive: false });
        return () => {
            canvas.removeEventListener('wheel', handleWheel);
        };
    }, [handleWheel]);

    return (
        <div style={{ position: 'relative', width: '100%', height: '100%', aspectRatio: '1 / 1' }}>
            <canvas
                width={1000}
                height = {1000}
                ref={canvasRef}
                onClick={handleClick}
                onMouseDown={handleMouseDown}
                onMouseMove={handleMouseMove}
                onMouseUp={handleMouseUp}
                onMouseLeave={() => {
                    setIsDragging(false);
                    setHoveredPixel(null);
                }}
                style={{
                    width: '100%',
                    height: '100%',
                    cursor: isDragging ? 'grabbing' : 'crosshair',
                    overscrollBehavior: 'none',
                    touchAction: 'none'
                }}
            />
        </div>
    );
});

PixelGrid.propTypes = {
    grid: PropTypes.object.isRequired,
    onPixelClick: PropTypes.func.isRequired,
    size: PropTypes.number.isRequired,
    colors: PropTypes.arrayOf(PropTypes.string).isRequired,
    connectedClients: PropTypes.number.isRequired,
};

export default PixelGrid;