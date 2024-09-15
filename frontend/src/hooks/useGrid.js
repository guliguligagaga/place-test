import { useCallback, useState, useEffect, useReducer } from 'react';

const GRID_SIZE = 100;

const gridReducer = (state, action) => {
    switch (action.type) {
        case 'UPDATE_CELL':
            const newGrid = new Uint8Array(state);
            const index = action.y * GRID_SIZE + action.x;
            newGrid[index] = action.colorIndex;
            return newGrid;
        case 'SET_GRID':
            return action.grid;
        default:
            return state;
    }
};

const useGrid = () => {
    const [grid, dispatch] = useReducer(gridReducer, new Uint8Array(GRID_SIZE * GRID_SIZE));

    useEffect(() => {
        console.log('Grid initialized with size:', grid.length);
    }, [grid]);

    const updateGrid = useCallback((x, y, colorIndex) => {
        dispatch({ type: 'UPDATE_CELL', x, y, colorIndex });
        console.log(`Grid updated at (${x}, ${y}) with color ${colorIndex}`);
    }, []);

    const setNewGrid = useCallback((newGrid) => {
        console.log('Setting new grid with size:', newGrid.length);
        dispatch({ type: 'SET_GRID', grid: newGrid });
    }, []);

    return [grid, setNewGrid, updateGrid];
};

export default useGrid;