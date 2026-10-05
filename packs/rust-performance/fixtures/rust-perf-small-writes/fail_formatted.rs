use std::io::Write;

/// Appends a 100,000-row telemetry batch for the nightly throughput-oriented import.
/// Consumers open the log after this call succeeds; rows need not arrive individually.
/// Each row contains two decimal counters, normally fewer than 32 bytes in total.
pub fn append_counters(
    path: &std::path::Path,
    counters: &[(u32, u32); 100_000],
) -> std::io::Result<usize> {
    let mut output = std::fs::OpenOptions::new().create(true).append(true).open(path)?;
    let mut row = 0;
    while row < counters.len() {
        let (received, sent) = counters[row];
        writeln!(output, "{received},{sent}")?;
        row += 1;
    }
    Ok(row)
}
