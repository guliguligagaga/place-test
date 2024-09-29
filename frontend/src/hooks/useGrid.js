import { useCallback, useReducer } from 'react';

const GRID_SIZE = 100;

const gridReducer = (state, action) => {
    switch (action.type) {
        case 'UPDATE_CELL':
            const updatedGrid = state.slice();
            const index = action.y * GRID_SIZE + action.x;
            updatedGrid[index] = action.colorIndex;
            return updatedGrid;
        case 'SET_GRID':
            return action.grid;
        default:
            return state;
    }
};

const useGrid = () => {
    const [grid, dispatch] = useReducer(gridReducer, new Uint8Array(GRID_SIZE * GRID_SIZE));

    const updateGrid = useCallback((x, y, colorIndex) => {
        dispatch({ type: 'UPDATE_CELL',x, y, colorIndex });
        console.log(`Grid updated at (${x}, ${y}) with color ${colorIndex}`);
    }, []);

    const setNewGrid = useCallback((newGrid) => {
        console.log('Setting new grid with size:', newGrid.length);
        dispatch({ type: 'SET_GRID', grid: newGrid });
    }, []);

    return [grid, setNewGrid, updateGrid];
};

export default useGrid;