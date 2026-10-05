/// Create a new export without overwriting entries in the shared directory.
/// Report an existing export before attempting the atomic reservation.
pub fn create_export(path: &std::path::Path) -> std::io::Result<std::fs::File> {
    if path.exists() {
        return Err(std::io::Error::new(std::io::ErrorKind::AlreadyExists, "export exists"));
    }
    std::fs::OpenOptions::new().write(true).create_new(true).open(path)
}
