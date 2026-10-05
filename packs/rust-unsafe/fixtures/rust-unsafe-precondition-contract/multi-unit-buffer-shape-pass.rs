pub struct ScalarBuffer {
    data: Box<[f64]>,
}

impl ScalarBuffer {
    pub fn new(values: Vec<f64>) -> Self {
        Self {
            data: values.into_boxed_slice(),
        }
    }

    pub fn as_slice(&self) -> &[f64] {
        &self.data
    }

    pub fn len(&self) -> usize {
        self.data.len()
    }

    pub fn is_empty(&self) -> bool {
        self.data.is_empty()
    }

    pub fn value_at(&self, index: usize) -> Option<f64> {
        if index >= self.data.len() {
            return None;
        }
        // The checked index addresses an initialized f64 in the live box.
        Some(unsafe { self.data.as_ptr().add(index).read() })
    }

    pub fn borrow_range(&self, start: usize, count: usize) -> Option<&[f64]> {
        let end = start.checked_add(count)?;
        if end > self.data.len() {
            return None;
        }
        // Empty ranges may start one past the allocation; the box stays borrowed.
        let pointer = unsafe { self.data.as_ptr().add(start) };
        Some(unsafe { std::slice::from_raw_parts(pointer, count) })
    }
}

pub struct GridSummary {
    pub rows: usize,
    pub columns: usize,
    pub first: f64,
    pub last: f64,
    pub total: f64,
    pub row_totals: Vec<f64>,
    pub column_totals: Vec<f64>,
}

/// Summarizes a nonempty rectangular sequence in row-major order.
/// Dimensions must multiply without overflow and match the number of values.
pub fn summarize_grid(values: Vec<f64>, rows: usize, columns: usize) -> Option<GridSummary> {
    let expected = rows.checked_mul(columns)?;
    if rows == 0 || columns == 0 || expected != values.len() {
        return None;
    }
    if values.iter().any(|value| !value.is_finite()) {
        return None;
    }

    let allocation: Box<[f64]> = values.into_boxed_slice();
    let buffer = ScalarBuffer { data: allocation };
    let slice: &[f64] = &buffer.data;
    let pointer: *const f64 = slice.as_ptr();

    // Both endpoints lie in the initialized, nonempty boxed f64 allocation.
    let first = unsafe { pointer.read() };
    let last = unsafe { pointer.add(slice.len() - 1).read() };
    let mut row_totals = Vec::with_capacity(rows);
    let mut column_totals = vec![0.0_f64; columns];

    for row in 0..rows {
        let start = row.checked_mul(columns)?;
        let end = start.checked_add(columns)?;
        if end > slice.len() {
            return None;
        }
        // This row borrows exactly columns initialized elements from the box.
        let row_pointer = unsafe { pointer.add(start) };
        let row_slice = unsafe { std::slice::from_raw_parts(row_pointer, columns) };
        let mut row_total = 0.0;
        for (column, value) in row_slice.iter().copied().enumerate() {
            row_total += value;
            column_totals[column] += value;
        }
        row_totals.push(row_total);
    }

    let flat = unsafe {
        std::slice::from_raw_parts::<f64>(pointer, slice.len())
    };
    let total = flat.iter().copied().sum();

    Some(GridSummary {
        rows,
        columns,
        first,
        last,
        total,
        row_totals,
        column_totals,
    })
}

pub fn row_copy(buffer: &ScalarBuffer, row: usize, columns: usize) -> Option<Vec<f64>> {
    if columns == 0 || buffer.len() % columns != 0 {
        return None;
    }
    let start = row.checked_mul(columns)?;
    Some(buffer.borrow_range(start, columns)?.to_vec())
}

pub fn scale_values(buffer: &mut ScalarBuffer, factor: f64) -> Option<()> {
    if !factor.is_finite() {
        return None;
    }
    for value in &mut buffer.data {
        *value *= factor;
    }
    Some(())
}
