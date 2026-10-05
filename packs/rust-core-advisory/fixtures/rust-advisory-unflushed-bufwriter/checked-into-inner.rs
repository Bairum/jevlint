use std::fs::File;
use std::io::{self, BufWriter, Write};
use std::path::Path;

pub fn write_receipt(path: &Path, receipt: &[u8]) -> io::Result<File> {
    let mut writer = BufWriter::new(File::create(path)?);
    writer.write_all(receipt)?;
    writer.into_inner().map_err(|error| error.into_error())
}
