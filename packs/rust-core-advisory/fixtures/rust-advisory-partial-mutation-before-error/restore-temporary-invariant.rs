use std::num::ParseIntError;

pub struct Download {
    pub expected_size: u64,
    pub changing_size: bool,
}

/// The update marker is temporary. Every return leaves changing_size false.
/// A failed conversion preserves the previous expected size.
pub fn update_size(download: &mut Download, size: &str) -> Result<(), ParseIntError> {
    download.changing_size = true;
    let outcome = (|| -> Result<u64, ParseIntError> { Ok(size.parse()?) })();
    download.changing_size = false;
    download.expected_size = outcome?;
    Ok(())
}
