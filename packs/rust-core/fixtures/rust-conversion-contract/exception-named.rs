/// Returns the low eight bits of the input.
pub fn truncate_low_byte(value: u32) -> u8 {
    value as u8
}

/// Rounds nanoseconds down to a whole number of milliseconds.
pub fn round_milliseconds_down(nanoseconds: u64) -> u64 {
    nanoseconds / 1_000_000
}
