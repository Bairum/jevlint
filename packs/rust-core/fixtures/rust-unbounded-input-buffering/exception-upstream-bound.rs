/// The envelope is already in memory. Enforce its maximum before the decoder
/// buffers it; std Cursor exposes only this checked slice, never the live peer.
pub fn decode_envelope(envelope: &[u8]) -> std::io::Result<Vec<u8>> {
    if envelope.len() > 4096 {
        return Err(std::io::Error::new(std::io::ErrorKind::InvalidData, "envelope too large"));
    }
    let mut input = std::io::Cursor::new(envelope);
    let mut decoded = Vec::new();
    std::io::Read::read_to_end(&mut input, &mut decoded)?;
    Ok(decoded)
}
