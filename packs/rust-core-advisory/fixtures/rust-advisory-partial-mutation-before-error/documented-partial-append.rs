use std::num::ParseIntError;

/// Appends values in order. On an error, all successfully parsed values before
/// the failing field remain appended, and the existing prefix is unchanged.
pub fn append_measurements(
    measurements: &mut Vec<u32>,
    fields: &[&str],
) -> Result<usize, ParseIntError> {
    let start = measurements.len();
    for field in fields {
        measurements.push(field.parse()?);
    }
    Ok(measurements.len() - start)
}
