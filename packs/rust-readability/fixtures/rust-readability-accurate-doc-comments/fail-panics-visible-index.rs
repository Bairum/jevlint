/// Returns the selected sensor reading.
///
/// # Panics
/// Never panics, including when the index is outside the readings slice.
pub fn selected_reading(readings: &[i32], index: usize) -> i32 {
    readings[index]
}
