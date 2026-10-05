use std::fs::File;
use std::io::{self, BufWriter as BufferedWriter, Write};
use std::path::Path;

pub fn begin_archive(path: &Path, version: u32) -> io::Result<BufferedWriter<File>> {
    let mut writer = BufferedWriter::new(File::create(path)?);
    writer.write_all(b"archive-version=")?;
    writeln!(writer, "{version}")?;
    Ok(writer)
}
