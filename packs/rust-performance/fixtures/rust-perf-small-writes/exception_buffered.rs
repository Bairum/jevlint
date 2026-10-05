/// Appends a million-byte batch to a caller-owned export writer.
/// The caller keeps the writer across throughput-oriented batches.
/// The importing worker may consume the batch only after successful completion.
/// Success includes delivery of previously queued bytes, not crash durability.
pub fn append_batch(
    output: &mut std::io::BufWriter<std::fs::File>,
    bytes: &[u8; 1_000_000],
) -> std::io::Result<usize> {
    for byte in bytes {
        <std::io::BufWriter<std::fs::File> as std::io::Write>::write_all(
            output,
            &[*byte],
        )?;
    }
    <std::io::BufWriter<std::fs::File> as std::io::Write>::flush(output)?;
    Ok(bytes.len())
}
