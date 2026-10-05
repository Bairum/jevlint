use std::fs::File;
use std::io::{self, BufWriter, Write};
use std::path::Path;

pub fn write_index(path: &Path, names: &[String]) -> io::Result<()> {
    let mut writer = BufWriter::new(File::create(path)?);
    for name in names {
        writeln!(writer, "{name}")?;
    }
    let _ = writer.flush();
    Ok(())
}
