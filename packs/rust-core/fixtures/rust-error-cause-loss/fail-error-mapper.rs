#[derive(Debug)]
pub enum DecodeError {
    InvalidLength,
    UnsupportedVersion,
    ChecksumMismatch,
}

#[derive(Debug, PartialEq)]
pub enum ImportError {
    InvalidLength,
    UnsupportedVersion,
    ChecksumMismatch,
}

/// Preserve the decoding failure category so callers can distinguish updating
/// their format version from repairing a damaged or truncated document.
pub fn map_decode_error(error: DecodeError) -> ImportError {
    match error {
        DecodeError::InvalidLength | DecodeError::UnsupportedVersion => ImportError::InvalidLength,
        DecodeError::ChecksumMismatch => ImportError::ChecksumMismatch,
    }
}
