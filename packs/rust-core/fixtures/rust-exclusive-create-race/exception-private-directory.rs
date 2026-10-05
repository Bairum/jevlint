/// The caller owns this newly created directory exclusively for this call;
/// no other actor can create or modify entries until ownership is released.
/// Create the first report, refusing to replace an earlier local report.
pub fn create_private_report(directory: &std::path::Path) -> std::io::Result<std::fs::File> {
    let path = directory.join("report.txt");
    if path.exists() {
        return Err(std::io::Error::new(std::io::ErrorKind::AlreadyExists, "report exists"));
    }
    std::fs::File::create(path)
}
