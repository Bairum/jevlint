use std::io;
use std::path::Path;

#[derive(Debug)]
pub enum OpenFailure {
    Missing,
    Io(io::Error),
}

/// Missing optional configuration selects defaults; all other failures are
/// returned with their original I/O identity for the caller's recovery policy.
pub fn open_optional_config(path: &Path) -> Result<std::fs::File, OpenFailure> {
    std::fs::File::open(path).map_err(|cause| match cause.kind() {
        std::io::ErrorKind::NotFound => OpenFailure::Missing,
        _ => OpenFailure::Io(cause),
    })
}
