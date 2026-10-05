use std::num::ParseIntError;

pub fn append_measurements(
    measurements: &mut Vec<u32>,
    fields: &[&str],
) -> Result<(), ParseIntError> {
    let original_len = measurements.len();
    let outcome = (|| -> Result<(), ParseIntError> {
        for field in fields {
            measurements.push(field.parse()?);
        }
        Ok(())
    })();
    if outcome.is_err() {
        measurements.truncate(original_len);
    }
    outcome
}
