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
