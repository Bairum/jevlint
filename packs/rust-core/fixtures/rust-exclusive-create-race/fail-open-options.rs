/// Reserve a new job file in a shared directory. Existing entries must
/// retain their contents, including entries created by concurrent workers.
pub fn reserve_job(path: &std::path::Path) -> std::io::Result<std::fs::File> {
    if std::fs::symlink_metadata(path).is_ok() {
        return Err(std::io::Error::new(std::io::ErrorKind::AlreadyExists, "job exists"));
    }
    std::fs::OpenOptions::new()
        .write(true)
        .create(true)
        .truncate(true)
        .open(path)
}
