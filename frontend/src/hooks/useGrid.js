import { useCallback, useState } from 'react';

const GRID_SIZE = 500;

const useGrid = () => {
    const [grid, setGrid] = useState(() => new Uint8Array(GRID_SIZE * GRID_SIZE));

    const updateGrid = useCallback((x, y, colorIndex) => {
        setGrid(prevGrid => {
            const newGrid = new Uint8Array(prevGrid);
            const index = y * GRID_SIZE + x;
            newGrid[index] = colorIndex;
            return newGrid;
        });
    }, []);

    return [grid, setGrid, updateGrid];
};

export default useGrid;