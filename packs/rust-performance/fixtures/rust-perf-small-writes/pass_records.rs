/// Exports 100,000 preencoded 16-byte records for throughput-oriented bulk import.
/// The importing worker opens the file only after successful completion.
/// Individual records need not become visible during the export.
/// Success reports the record count, not crash durability.
pub fn export_records(
    path: &std::path::Path,
    records: &[[u8; 16]; 100_000],
) -> std::io::Result<usize> {
    let file: std::fs::File = std::fs::File::create(path)?;
    let mut output: std::io::BufWriter<std::fs::File> = std::io::BufWriter::new(file);
    for record in records {
        <std::io::BufWriter<std::fs::File> as std::io::Write>::write_all(
            &mut output,
            record,
        )?;
    }
    <std::io::BufWriter<std::fs::File> as std::io::Write>::flush(&mut output)?;
    Ok(records.len())
}
