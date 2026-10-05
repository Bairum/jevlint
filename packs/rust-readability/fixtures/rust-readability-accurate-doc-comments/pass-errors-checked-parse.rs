/// Parses the retry count from a configuration value.
///
/// # Errors
/// Returns an error for invalid integer text or a value outside the u16 range.
pub fn parse_retry_count(text: &str) -> Result<u16, std::num::ParseIntError> {
    let retries = text.parse::<u16>()?;
    Ok(retries)
}
