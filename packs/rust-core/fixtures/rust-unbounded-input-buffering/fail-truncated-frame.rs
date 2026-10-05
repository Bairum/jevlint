/// Remote EOF-delimited requests must be complete and no more than 4096 bytes;
/// oversized messages must be rejected rather than silently accepted as a prefix.
pub fn read_request(stream: &mut std::net::TcpStream) -> std::io::Result<Vec<u8>> {
    let mut limited = std::io::Read::take(stream, 4096);
    let mut bytes = Vec::new();
    std::io::Read::read_to_end(&mut limited, &mut bytes)?;
    Ok(bytes)
}
