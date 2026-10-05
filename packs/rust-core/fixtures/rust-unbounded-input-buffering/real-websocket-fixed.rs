/// Decode one untrusted WebSocket binary frame after its opcode and flags have
/// been validated. The extended length is an eight-byte network-order integer.
/// The application frame_limit caps payload allocation; incomplete frames are errors.
pub fn read_dataframe(
    stream: &mut std::net::TcpStream,
    frame_limit: usize,
) -> std::io::Result<Vec<u8>> {
    let mut extended_length = [0_u8; 8];
    std::io::Read::read_exact(stream, &mut extended_length)?;
    let length = usize::try_from(u64::from_be_bytes(extended_length))
        .map_err(|_| std::io::Error::new(std::io::ErrorKind::InvalidData, "invalid length"))?;
    if length > frame_limit {
        return Err(std::io::Error::new(std::io::ErrorKind::InvalidData, "frame too large"));
    }
    let mut payload = Vec::with_capacity(length);
    let mut frame = std::io::Read::take(stream, length as u64);
    std::io::Read::read_to_end(&mut frame, &mut payload)?;
    if payload.len() != length {
        return Err(std::io::Error::new(std::io::ErrorKind::UnexpectedEof, "incomplete frame"));
    }
    Ok(payload)
}
