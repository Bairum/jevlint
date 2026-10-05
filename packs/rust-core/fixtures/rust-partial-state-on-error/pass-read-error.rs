/// Append one network chunk. On Err, output and the stream's unread bytes
/// remain available to the caller; on Ok, append the returned prefix.
pub fn append_one_read(reader: &mut std::net::TcpStream, output: &mut Vec<u8>) -> std::io::Result<usize> {
    let mut chunk = [0u8; 64];
    let count = std::io::Read::read(reader, &mut chunk)?;
    output.extend_from_slice(&chunk[..count]);
    Ok(count)
}
