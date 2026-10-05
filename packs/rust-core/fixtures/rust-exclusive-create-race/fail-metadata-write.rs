/// Publish a new receipt without replacing another process's receipt.
/// The receipt directory is shared by concurrent workers.
pub fn publish_receipt(path: &std::path::Path, receipt: &[u8]) -> std::io::Result<()> {
    match std::fs::metadata(path) {
        Ok(_) => Err(std::io::Error::new(std::io::ErrorKind::AlreadyExists, "receipt exists")),
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
            std::fs::write(path, receipt)
        }
        Err(error) => Err(error),
    }
}
