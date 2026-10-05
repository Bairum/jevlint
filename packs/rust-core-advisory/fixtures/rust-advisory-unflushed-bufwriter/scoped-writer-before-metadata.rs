use std::fs::{self, File};
use std::io::{self, BufWriter, Write};
use std::path::Path;

pub fn write_catalog(path: &Path, entries: &[String]) -> io::Result<u64> {
    {
        let mut writer = BufWriter::new(File::create(path)?);
        writeln!(writer, "entries={}", entries.len())?;
        for entry in entries {
            writeln!(writer, "{entry}")?;
        }
    }
    Ok(fs::metadata(path)?.len())
}
