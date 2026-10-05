use std::fs::File;
use std::io::{self, Write};
use std::path::Path;

pub fn initialize_credentials(path: &Path, credentials: &[u8]) -> io::Result<()> {
    if path.exists() {
        return Err(io::Error::new(io::ErrorKind::AlreadyExists, "credentials exist"));
    }
    let mut file = File::create(path)?;
    file.write_all(credentials)?;
    file.sync_all()
}
