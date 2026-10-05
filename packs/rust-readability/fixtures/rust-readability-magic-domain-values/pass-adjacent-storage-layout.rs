use std::mem::size_of;

pub struct SolverScratch {
    // Three full-size buffers: one value per cell, including both endpoints.
    pub current: Vec<f64>,
    pub next: Vec<f64>,
    pub residual: Vec<f64>,
    // Two interior buffers: omit the first and last cells from each buffer.
    pub lower_diagonal: Vec<f64>,
    pub upper_diagonal: Vec<f64>,
}

impl SolverScratch {
    /// Bytes occupied by the five buffers' elements, excluding Vec headers
    /// and allocator overhead. The 3 and 2 count the declared buffers above;
    /// subtracting 2 removes the endpoint cells from each interior buffer.
    pub fn required_bytes(cells: usize) -> Option<usize> {
        let full_elements = cells.checked_mul(3)?;
        let interior_elements = cells.saturating_sub(2).checked_mul(2)?;
        full_elements
            .checked_add(interior_elements)?
            .checked_mul(size_of::<f64>())
    }

    pub fn allocate(cells: usize) -> Self {
        let interior_cells = cells.saturating_sub(2);
        Self {
            current: vec![0.0; cells],
            next: vec![0.0; cells],
            residual: vec![0.0; cells],
            lower_diagonal: vec![0.0; interior_cells],
            upper_diagonal: vec![0.0; interior_cells],
        }
    }
}
