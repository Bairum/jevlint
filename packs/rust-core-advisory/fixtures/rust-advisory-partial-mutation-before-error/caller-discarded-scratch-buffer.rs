use std::num::ParseIntError;

/// Fills per-call scratch, not retained application data. The caller drops this
/// buffer on Err; partially parsed readings are never published.
fn parse_readings(scratch: &mut Vec<u32>, fields: &[&str]) -> Result<(), ParseIntError> {
    scratch.clear();
    for field in fields {
        scratch.push(field.parse()?);
    }
    Ok(())
}

pub fn distinct_reading_count(fields: &[&str]) -> Result<usize, ParseIntError> {
    let mut scratch = Vec::with_capacity(fields.len());
    parse_readings(&mut scratch, fields)?;
    scratch.sort_unstable();
    scratch.dedup();
    Ok(scratch.len())
}
