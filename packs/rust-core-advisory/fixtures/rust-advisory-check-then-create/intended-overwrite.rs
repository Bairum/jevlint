use std::fs;
use std::io;
use std::path::Path;

/// Replaces the cached report, including reports published by other workers.
/// The returned flag records whether a report was observed before this write.
pub fn replace_report(path: &Path, contents: &[u8]) -> io::Result<bool> {
    let previously_observed = path.exists();
    fs::write(path, contents)?;
    Ok(previously_observed)
}
