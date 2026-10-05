use std::num::ParseIntError;

/// Writes into a fresh, unpublished output buffer. The caller discards the
/// entire buffer on Err and returns it only after all readings are encoded.
fn encode_into(output: &mut Vec<u8>, fields: &[&str]) -> Result<(), ParseIntError> {
    output.push(1); // Format version.
    for field in fields {
        let reading: u16 = field.parse()?;
        output.extend_from_slice(&reading.to_le_bytes());
    }
    Ok(())
}

pub fn encode_readings(fields: &[&str]) -> Result<Vec<u8>, ParseIntError> {
    let mut output = Vec::new();
    encode_into(&mut output, fields)?;
    Ok(output)
}
