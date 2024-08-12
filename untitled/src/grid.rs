use std::time::SystemTime;
use serde::{Deserialize, Serialize};
use rand::Rng;

const COLOR_PALETTE: [(u8, u8, u8); 16] = [
    (0, 0, 0),        // 0: Black
    (255, 0, 0),      // 1: Red
    (0, 255, 0),      // 2: Green
    (255, 255, 0),    // 3: Yellow
    (0, 0, 255),      // 4: Blue
    (255, 0, 255),    // 5: Magenta
    (0, 255, 255),    // 6: Cyan
    (255, 255, 255),  // 7: White
    (128, 0, 0),      // 8: Maroon
    (255, 165, 0),    // 9: Orange
    (255, 192, 203),  // 10: Pink
    (128, 128, 0),    // 11: Olive
    (0, 128, 128),    // 12: Teal
    (128, 0, 128),    // 13: Purple
    (192, 192, 192),  // 14: Silver
    (128, 128, 128),  // 15: Gray
];

const WIDTH: usize = 500;
const HEIGHT: usize = 500;

#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub struct Cell {
    color: u8,
    timestamp: u32,
}

impl Cell {
    fn new(color: u8, timestamp: u32) -> Self {
        Cell { color, timestamp }
    }
}
pub struct Grid {
    pub cells: Vec<Cell>,
}

impl Grid {
    pub(crate) fn new() -> Self {
        let mut rng = rand::thread_rng();
        let mut cells = Vec::with_capacity(WIDTH * HEIGHT);

        for _ in 0..(WIDTH * HEIGHT) {
            cells.push(Cell {
                color: rng.gen_range(0..16),
                timestamp: 0,
            });
        }

        Grid { cells }
    }

    pub(crate) fn modify_cell(&mut self, x: usize, y: usize, color: u8) -> Option<()> {
        if x < WIDTH && y < HEIGHT {
            let index = y * WIDTH + x;
            let timestamp = SystemTime::now().duration_since(SystemTime::UNIX_EPOCH).ok()?.as_millis() as u32;
            self.cells[index] = Cell::new(color, timestamp);
            Some(())
        } else {
            None
        }
    }

    pub fn get_cell(&self, x: usize, y: usize) -> Option<&Cell> {
        if x < WIDTH && y < HEIGHT {
            let index = y * WIDTH + x;
            Some(&self.cells[index])
        } else {
            None
        }
    }

    pub fn to_bitfield(&self) -> Vec<u8> {
        let mut bitfield = vec![0u8; WIDTH * HEIGHT / 2];

        for (index, cell) in self.cells.iter().enumerate() {
            let byte_index = index / 2;
            let bit_offset = (index % 2) * 4;

            let byte = &mut bitfield[byte_index];
            *byte &= !(0xF << bit_offset);
            *byte |= cell.color << bit_offset;
        }

        bitfield
    }
}