import { useCallback, useReducer } from 'react';

const GRID_SIZE = 100;

const gridReducer = (state, action) => {
    switch (action.type) {
        case 'UPDATE_CELL':
            const updatedGrid = state.slice();
            const index = action.y * GRID_SIZE + action.x;
            //console.log('Index:', index);
            //console.log('x: y: colorIndex:', action.x, action.y, action.colorIndex);
            updatedGrid[index] = action.colorIndex;
            //console.log('Updated Grid:', updatedGrid);
            return updatedGrid;
        case 'SET_GRID':
            const uint8Array = new Uint8Array(action.grid);
            const unpackedGrid = new Uint8Array(GRID_SIZE * GRID_SIZE);

            for (let y = 0; y < GRID_SIZE; y++) {
                for (let x = 0; x < GRID_SIZE; x++) {
                    const byteIndex = Math.floor((y * GRID_SIZE + x) / 2);
                    const isUpperNibble = x % 2 === 0;
                    const byte = uint8Array[byteIndex];

                    let color;
                    if (isUpperNibble) {
                        color = (byte & 0xF0) >> 4;
                    } else {
                        color = byte & 0x0F;
                    }

                    unpackedGrid[y * GRID_SIZE + x] = color;
                }
            }
            return unpackedGrid;
        default:
            return state;
    }
};

const useGrid = () => {
    const [grid, dispatch] = useReducer(gridReducer, new Uint8Array(GRID_SIZE * GRID_SIZE));

    const updateGrid = useCallback((x, y, colorIndex) => {
        dispatch({ type: 'UPDATE_CELL',x, y, colorIndex });
    }, []);

    const setGrid = useCallback((newGrid) => {
        dispatch({ type: 'SET_GRID', grid: newGrid });
    }, []);

    return [grid, setGrid, updateGrid];
};

export default useGrid;