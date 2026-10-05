/// Save the diagnose output without replacing an existing manifest.
/// Multiple CLI processes can write to the same output directory.
pub fn save_diagnose(path: &std::path::Path, manifest: &[u8]) -> std::io::Result<()> {
    if path.exists() {
        return Err(std::io::Error::new(std::io::ErrorKind::AlreadyExists, "output exists"));
    }
    let mut output = std::fs::File::create(path)?;
    std::io::Write::write_all(&mut output, manifest)
}
