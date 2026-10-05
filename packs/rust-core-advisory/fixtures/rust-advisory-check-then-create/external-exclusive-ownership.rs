use std::fs;
use std::io;
use std::path::{Component, Path};

/// The caller holds the deployment controller's exclusive directory lease for
/// the entire call. The controller grants only one process access to this
/// directory, including its parent entry, until that lease is released.
/// No other process may create, remove, or rename entries during the lease.
pub fn initialize_leased_output(
    leased_directory: &Path,
    name: &str,
    contents: &[u8],
) -> io::Result<bool> {
    let relative = Path::new(name);
    let mut components = relative.components();
    if !matches!(components.next(), Some(Component::Normal(_)))
        || components.next().is_some()
    {
        return Err(io::Error::new(io::ErrorKind::InvalidInput, "expected one filename"));
    }
    let path = leased_directory.join(relative);
    match fs::metadata(&path) {
        Ok(_) => Ok(false),
        Err(error) if error.kind() == io::ErrorKind::NotFound => {
            fs::write(&path, contents)?;
            Ok(true)
        }
        Err(error) => Err(error),
    }
}
