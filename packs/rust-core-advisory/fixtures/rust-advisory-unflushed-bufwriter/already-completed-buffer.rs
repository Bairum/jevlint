use std::fs::File;
use std::io::{self, BufWriter, Write};
use std::path::Path;

pub fn write_report(path: &Path, report: &[u8]) -> io::Result<u64> {
    let mut writer = BufWriter::new(File::create(path)?);
    writer.write_all(report)?;
    writer.flush()?;
    let stored_bytes = writer.get_ref().metadata()?.len();
    drop(writer);
    Ok(stored_bytes)
}
