#[derive(Debug)]
pub enum FrameError {
    UnsupportedVersion,
}

pub fn parse_remote_frame(bytes: &[u8]) -> Result<&[u8], FrameError> {
    assert!(bytes.len() >= 2, "frame needs a version and flags");
    if bytes[0] != 1 {
        return Err(FrameError::UnsupportedVersion);
    }
    Ok(&bytes[2..])
}
