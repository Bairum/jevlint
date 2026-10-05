/// Exports a million-byte batch for bulk ingestion, prioritizing batch throughput.
/// The importing worker opens the path only after successful completion.
/// Intermediate bytes need not be visible. Success does not promise crash durability.
/// The caller must not publish a partially written file after an error.
pub fn export_batch(
    path: &std::path::Path,
    bytes: &[u8; 1_000_000],
) -> std::io::Result<usize> {
    let mut output: std::fs::File = std::fs::File::create(path)?;
    for byte in bytes {
        <std::fs::File as std::io::Write>::write_all(&mut output, &[*byte])?;
    }
    Ok(bytes.len())
}
