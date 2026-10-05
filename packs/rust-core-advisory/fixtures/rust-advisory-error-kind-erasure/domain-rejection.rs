use std::path::Path;

#[derive(Debug)]
pub struct Rejected;

/// Admission requires a readable local permit whose entire contents are
/// "approved". An absent, unreadable or malformed permit denies admission;
/// callers need only the admission decision, not storage diagnostics.
pub fn admit(path: &Path) -> Result<(), Rejected> {
    let permit = std::fs::read_to_string(path).map_err(|_| Rejected)?;
    if permit.trim() == "approved" {
        Ok(())
    } else {
        Err(Rejected)
    }
}
