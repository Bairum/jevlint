use std::convert::Infallible;
use std::path::Path;

/// The probe API uses Result for all providers. This local provider is
/// infallible: unavailable, unreadable and absent resources all return false.
pub fn is_available(path: &Path) -> Result<bool, Infallible> {
    let readable = std::fs::File::open(path).map_err(|_| ()).is_ok();
    Ok(readable)
}
