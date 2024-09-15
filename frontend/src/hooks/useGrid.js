import { useCallback, useState, useEffect } from 'react';

const GRID_SIZE = 100;

const useGrid = () => {
    const [grid, setGrid] = useState(() => new Uint8Array(GRID_SIZE * GRID_SIZE));

    useEffect(() => {
        console.log('Grid initialized with size:', grid.length);
    }, [grid]);

    const updateGrid = useCallback((x, y, colorIndex) => {
        setGrid(prevGrid => {
            const newGrid = new Uint8Array(prevGrid);
            const index = y * GRID_SIZE + x;
            newGrid[index] = colorIndex;
            console.log(`Grid updated at (${x}, ${y}) with color ${colorIndex}`);
            return newGrid;
        });
    }, []);

    const setNewGrid = useCallback((newGrid) => {
        console.log('Setting new grid with size:', newGrid.length);
        setGrid(newGrid);
    }, []);

    return [grid, setNewGrid, updateGrid];
};

export default useGrid;