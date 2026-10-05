use std::fs::OpenOptions;
use std::io::{self, BufWriter};
use std::path::Path;

pub fn ensure_report_exists(path: &Path) -> io::Result<()> {
    let file = OpenOptions::new().create(true).append(true).open(path)?;
    let _writer = BufWriter::new(file);
    Ok(())
}
