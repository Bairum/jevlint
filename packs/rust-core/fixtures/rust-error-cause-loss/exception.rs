/// Public response hides local paths and OS details. Write the resource and
/// original diagnostic to the internal audit stream before returning None.
/// Failure to persist that diagnostic remains an error for the internal caller.
pub fn public_config(path: &std::path::Path, audit: &mut std::fs::File) -> std::io::Result<Option<String>> {
    match std::fs::read_to_string(path) {
        Ok(text) => Ok(Some(text)),
        Err(error) => {
            let diagnostic = format!("config read at {}: {:?}\n", path.display(), error);
            std::io::Write::write_all(audit, diagnostic.as_bytes())?;
            Ok(None)
        }
    }
}
