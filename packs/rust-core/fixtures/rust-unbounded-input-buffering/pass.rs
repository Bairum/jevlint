/// A remote peer controls this EOF-delimited request. The protocol maximum is
/// 4096 bytes. Read at most one extra byte to reject oversize, not accept a prefix.
/// This is a byte bound, not a slow-peer timeout or global OOM guarantee.
pub fn read_request(stream: &mut std::net::TcpStream) -> std::io::Result<Vec<u8>> {
    let mut limited = std::io::Read::take(stream, 4097);
    let mut bytes = Vec::new();
    std::io::Read::read_to_end(&mut limited, &mut bytes)?;
    if bytes.len() > 4096 {
        return Err(std::io::Error::new(std::io::ErrorKind::InvalidData, "request too large"));
    }
    Ok(bytes)
}
