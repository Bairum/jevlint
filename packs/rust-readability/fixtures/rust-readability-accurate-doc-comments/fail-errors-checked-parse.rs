/// Parses the retry count from a configuration value.
///
/// # Errors
/// This function never returns an error, regardless of the input text.
pub fn parse_retry_count(text: &str) -> Result<u16, std::num::ParseIntError> {
    let retries = text.parse::<u16>()?;
    Ok(retries)
}
