import { useCallback, useState } from 'react';

const GRID_SIZE = 100;

const useGrid = () => {
    const [grid, setGrid] = useState(() => Array(GRID_SIZE * GRID_SIZE).fill('#FFFFFF'));

    const updateGrid = useCallback((index, color) => {
        setGrid(prevGrid => {
            const newGrid = [...prevGrid];
            newGrid[index] = color;
            return newGrid;
        });
    }, []);

    return [grid, setGrid, updateGrid];
};

export default useGrid;