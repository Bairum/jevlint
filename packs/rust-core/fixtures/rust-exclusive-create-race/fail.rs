/// Create an export without overwriting any existing file. Other processes
/// may create the same target concurrently in this shared export directory.
pub fn create_export(path: &std::path::Path) -> std::io::Result<std::fs::File> {
    if path.exists() {
        return Err(std::io::Error::new(std::io::ErrorKind::AlreadyExists, "export exists"));
    }
    std::fs::File::create(path)
}
