use std::fs::OpenOptions;
use std::io::{self, Write};
use std::path::Path;

pub fn claim_job(path: &Path, owner: &[u8]) -> io::Result<bool> {
    if path.exists() {
        return Ok(false);
    }
    let mut file = match OpenOptions::new().write(true).create_new(true).open(path) {
        Ok(file) => file,
        Err(error) if error.kind() == io::ErrorKind::AlreadyExists => return Ok(false),
        Err(error) => return Err(error),
    };
    file.write_all(owner)?;
    file.sync_all()?;
    Ok(true)
}
