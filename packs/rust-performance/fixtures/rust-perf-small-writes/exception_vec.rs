use std::io::Write;

/// Encodes 100,000 telemetry values in memory for a bulk compression stage.
/// The compressor receives the completed vector and needs no incremental delivery.
pub fn encode_counters(counters: &[u32; 100_000]) -> std::io::Result<Vec<u8>> {
    let mut output = Vec::with_capacity(counters.len() * 12);
    for counter in counters {
        write!(output, "{counter},")?;
    }
    Ok(output)
}

/// Serializes a million-byte frame before handing it to an in-memory checksum worker.
pub fn encode_frame(bytes: &[u8; 1_000_000]) -> std::io::Result<Vec<u8>> {
    let mut output = Vec::with_capacity(bytes.len());
    for byte in bytes {
        output.write_all(&[*byte])?;
    }
    Ok(output)
}
