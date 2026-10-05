use std::io;
use std::path::Path;

#[derive(Debug)]
pub struct ReadFailure {
    pub kind: io::ErrorKind,
    pub path: std::path::PathBuf,
}

/// Reads a template. Callers select recovery by kind and display its path;
/// platform-specific error text is not part of this interface.
pub fn read_template(path: &Path) -> Result<String, ReadFailure> {
    std::fs::read_to_string(path).map_err(|error| ReadFailure {
        kind: error.kind(),
        path: path.to_owned(),
    })
}
