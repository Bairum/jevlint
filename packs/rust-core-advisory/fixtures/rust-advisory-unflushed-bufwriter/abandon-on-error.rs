use std::fs::File;
use std::io::{self, BufWriter, Write};
use std::path::Path;

pub fn write_manifest(path: &Path, records: &[String]) -> io::Result<()> {
    let mut writer = BufWriter::new(File::create(path)?);
    writeln!(writer, "records={}", records.len())?;
    for record in records {
        if record.contains('\n') {
            return Err(io::Error::new(io::ErrorKind::InvalidInput, "multiline record"));
        }
        writeln!(writer, "{record}")?;
    }
    writer.flush()
}
