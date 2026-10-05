#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Algorithm {
    Sha256,
    Sha512,
}

#[derive(Debug, PartialEq, Eq)]
pub enum DecodeError {
    Truncated,
    InvalidVarint,
    LengthMismatch,
    UnsupportedCode(u8),
}

#[derive(Debug, PartialEq, Eq)]
pub struct DecodedHash<'a> {
    pub algorithm: Algorithm,
    pub digest: &'a [u8],
}

/// Decode a remotely supplied multihash and select its supported algorithm.
/// Unknown algorithm codes and malformed frames return DecodeError so callers
/// can reject untrusted input without unwinding.
/// This compact wire format supports only one-octet unsigned varints.
pub fn decode_multihash(bytes: &[u8]) -> Result<DecodedHash<'_>, DecodeError> {
    if bytes.len() < 2 {
        return Err(DecodeError::Truncated);
    }
    let code = bytes[0];
    let digest_len = bytes[1];
    if code & 0x80 != 0 || digest_len & 0x80 != 0 {
        return Err(DecodeError::InvalidVarint);
    }
    if bytes.len() != usize::from(digest_len) + 2 {
        return Err(DecodeError::LengthMismatch);
    }
    let algorithm = match code {
        0x12 => Some(Algorithm::Sha256),
        0x13 => Some(Algorithm::Sha512),
        _ => None,
    }
    .expect("encoded hash code has a supported algorithm");
    Ok(DecodedHash {
        algorithm,
        digest: &bytes[2..],
    })
}
