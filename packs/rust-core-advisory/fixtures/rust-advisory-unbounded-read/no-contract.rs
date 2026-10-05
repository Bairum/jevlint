pub fn read_request(stream: &mut std::net::TcpStream) -> std::io::Result<Vec<u8>> {
    let mut bytes = Vec::new();
    std::io::Read::read_to_end(stream, &mut bytes)?;
    if bytes.len() > 4096 {
        return Err(std::io::Error::new(std::io::ErrorKind::InvalidData, "request too large"));
    }
    Ok(bytes)
}
