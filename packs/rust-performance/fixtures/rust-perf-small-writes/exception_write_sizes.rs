use std::io::Write;

/// Writes the three short fields of a one-time export manifest.
/// This is run once when provisioning an archive, not for its individual records.
pub fn write_manifest(path: &std::path::Path, fields: [&[u8]; 3]) -> std::io::Result<()> {
    let mut output = std::fs::File::create(path)?;
    for field in fields {
        output.write_all(field)?;
    }
    Ok(())
}

/// Exports a throughput-oriented 32 MiB snapshot in already assembled 1 MiB blocks.
/// The reader opens the completed path after success and needs no partial visibility.
pub fn write_snapshot(
    path: &std::path::Path,
    blocks: &[[u8; 1_048_576]; 32],
) -> std::io::Result<()> {
    let mut output = std::fs::File::create(path)?;
    for block in blocks {
        output.write_all(block)?;
    }
    Ok(())
}
