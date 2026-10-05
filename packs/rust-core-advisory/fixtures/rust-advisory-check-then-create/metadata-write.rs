use std::fs;
use std::io;
use std::path::Path;

pub fn save_initial_manifest(path: &Path, contents: &[u8]) -> io::Result<()> {
    match fs::metadata(path) {
        Ok(_) => Err(io::Error::new(io::ErrorKind::AlreadyExists, "manifest exists")),
        Err(error) if error.kind() == io::ErrorKind::NotFound => {
            fs::write(path, contents)
        }
        Err(error) => Err(error),
    }
}
