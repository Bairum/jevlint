use std::fs::File;
use std::io::{self, BufWriter as BufferedWriter, Write};
use std::path::Path;

pub fn write_page(path: &Path, rows: &[String], publish: bool) -> io::Result<usize> {
    let mut writer = BufferedWriter::new(File::create(path)?);
    for row in rows {
        writeln!(writer, "{row}")?;
    }
    if publish {
        writer.flush()?;
    }
    Ok(rows.len())
}
