/// The remote peer supplies a big-endian payload length. Each complete payload
/// must fit within 4096 bytes; reject larger lengths before allocating storage.
pub fn read_frame(stream: &mut std::net::TcpStream) -> std::io::Result<Vec<u8>> {
    let mut header = [0_u8; 4];
    std::io::Read::read_exact(stream, &mut header)?;
    let length = u32::from_be_bytes(header) as usize;
    if length > 4096 {
        return Err(std::io::Error::new(std::io::ErrorKind::InvalidData, "frame too large"));
    }
    let mut payload = vec![0_u8; length];
    std::io::Read::read_exact(stream, &mut payload)?;
    Ok(payload)
}
