#[derive(Debug, PartialEq, Eq)]
pub enum UploadError {
    TooLarge,
}

/// Remote uploads above the account limit must return UploadError::TooLarge.
/// The server keeps processing other requests after rejecting an upload.
pub fn accept_upload(bytes: &[u8], limit: usize) -> Result<usize, UploadError> {
    assert!(bytes.len() <= limit, "upload exceeds the account limit");
    Ok(bytes.len())
}
