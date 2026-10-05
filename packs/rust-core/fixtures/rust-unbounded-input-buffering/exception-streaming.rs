/// Count an untrusted EOF-delimited payload without retaining it. The protocol
/// permits at most 4096 bytes and requires rejection of any larger payload.
pub fn count_payload(stream: &mut std::net::TcpStream) -> std::io::Result<usize> {
    let mut buffer = [0_u8; 512];
    let mut total = 0_usize;
    loop {
        let count = std::io::Read::read(stream, &mut buffer)?;
        if count == 0 {
            return Ok(total);
        }
        if count > 4096 - total {
            return Err(std::io::Error::new(std::io::ErrorKind::InvalidData, "payload too large"));
        }
        total += count;
    }
}
