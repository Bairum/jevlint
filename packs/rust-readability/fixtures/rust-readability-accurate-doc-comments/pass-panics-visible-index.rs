/// Returns the selected sensor reading.
///
/// # Panics
/// Panics when index is greater than or equal to readings.len().
pub fn selected_reading(readings: &[i32], index: usize) -> i32 {
    readings[index]
}
