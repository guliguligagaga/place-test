import React, { useRef, useState, useEffect, useCallback } from 'react';
import PropTypes from 'prop-types';

const INITIAL_ZOOM = 1;
const MAX_ZOOM = 40;
const MIN_ZOOM = 0.1;
const QUADRANT_SIZE = 32;

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

        // Draw quadrants
        ctx.strokeStyle = 'rgba(0, 0, 0, 0.2)';
        ctx.lineWidth = 1 / zoom;
        for (let x = 0; x <= size; x += QUADRANT_SIZE) {
            ctx.beginPath();
            ctx.moveTo(x * scaleFactor, 0);
            ctx.lineTo(x * scaleFactor, size * scaleFactor);
            ctx.stroke();
        }
        for (let y = 0; y <= size; y += QUADRANT_SIZE) {
            ctx.beginPath();
            ctx.moveTo(0, y * scaleFactor);
            ctx.lineTo(size * scaleFactor, y * scaleFactor);
            ctx.stroke();
        }

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

        // // Draw connected clients count
        // ctx.fillStyle = 'black';
        // ctx.font = '14px Arial';
        // ctx.textAlign = 'left';
        // ctx.textBaseline = 'top';
        // ctx.fillText(`Connected Clients: ${connectedClients}`, 10, 10);

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

    // const getQuadrantId = useCallback((x, y) => {
    //     const quadrant = quadrants.find(q => x >= q.x && x < q.x + QUADRANT_SIZE && y >= q.y && y < q.y + QUADRANT_SIZE);
    //     return quadrant ? quadrant.id : null;
    // }, [quadrants]);

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

            // const quadrantId = getQuadrantId(x, y);
            // if (quadrantId !== null && !subscribedQuadrants.has(quadrantId)) {
            //     onSubscribe(quadrantId);
            // }
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

    // const getVisibleQuadrants = useCallback(() => {
    //     const visibleQuadrants = [];
    //     const visibleWidth = size / zoom;
    //     const visibleHeight = size / zoom;
    //
    //     const startX = offset.x;
    //     const startY = offset.y;
    //     const endX = offset.x + visibleWidth;
    //     const endY = offset.y + visibleHeight;
    //
    //     quadrants.forEach(quadrant => {
    //         if (quadrant.x < endX && quadrant.x + QUADRANT_SIZE > startX &&
    //             quadrant.y < endY && quadrant.y + QUADRANT_SIZE > startY) {
    //             visibleQuadrants.push(quadrant.id);
    //         }
    //     });
    //
    //     return visibleQuadrants;
    // }, [offset, zoom, size, quadrants]);

    // useEffect(() => {
    //     const visibleQuadrants = getVisibleQuadrants();
    //     visibleQuadrants.forEach(quadrantId => {
    //         if (!subscribedQuadrants.has(quadrantId)) {
    //             onSubscribe(quadrantId);
    //         }
    //     });
    //     subscribedQuadrants.forEach(quadrantId => {
    //         if (!visibleQuadrants.includes(quadrantId)) {
    //             onUnsubscribe(quadrantId);
    //         }
    //     });
    // }, [offset, zoom, subscribedQuadrants, onSubscribe, onUnsubscribe, getVisibleQuadrants]);

    return (
        <div style={{ position: 'relative', width: '100%', height: '100%', aspectRatio: '1 / 1' }}>
            <canvas
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
    // quadrants: PropTypes.arrayOf(PropTypes.shape({
    //     id: PropTypes.number.isRequired,
    //     x: PropTypes.number.isRequired,
    //     y: PropTypes.number.isRequired,
    // })).isRequired,
    //subscribedQuadrants: PropTypes.instanceOf(Set).isRequired,
    //onSubscribe: PropTypes.func.isRequired,
    //onUnsubscribe: PropTypes.func.isRequired,
    connectedClients: PropTypes.number.isRequired,
};

export default PixelGrid;