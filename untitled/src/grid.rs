use rand::Rng;
use serde::{Deserialize, Serialize};
use std::time::SystemTime;

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
            let timestamp = SystemTime::now()
                .duration_since(SystemTime::UNIX_EPOCH)
                .ok()?
                .as_millis() as u32;
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
