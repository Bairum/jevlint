/// Parses a retry count.
///
/// # Errors
/// Returns a parse error when the text is not a valid `u16`.
pub fn parse_retry_count(text: &str) -> Result<u16, std::num::ParseIntError> {
    let retries = text.parse::<u16>()?;
    Ok(retries)
}

/// Returns the reading at `index`.
///
/// # Panics
/// Panics when `index` is outside the readings slice.
pub fn selected_reading(readings: &[i32], index: usize) -> i32 {
    readings[index]
}

/// Copies one sample from caller-supplied memory.
///
/// # Safety
/// `sample` must point to an initialized, aligned `u32` readable for this call.
/// The pointed-to value must not be concurrently mutated during the read.
pub unsafe fn copy_sample(sample: *const u32) -> u32 {
    unsafe { *sample }
}

/// Appends all supplied readings to the destination.
pub fn publish_readings(destination: &mut Vec<i32>, batch: &[i32]) {
    destination.extend_from_slice(batch);
}
