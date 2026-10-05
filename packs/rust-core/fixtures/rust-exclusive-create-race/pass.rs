/// Create a new export without overwriting any existing file, including
/// files concurrently created by another process in the shared directory.
pub fn create_export(path: &std::path::Path) -> std::io::Result<std::fs::File> {
    std::fs::OpenOptions::new().write(true).create_new(true).open(path)
}
