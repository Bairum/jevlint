/// Replace the current report and return whether an earlier report existed.
pub fn replace_report(path: &std::path::Path) -> std::io::Result<(std::fs::File, bool)> {
    let replaced = path.exists();
    let file = std::fs::File::create(path)?;
    Ok((file, replaced))
}
