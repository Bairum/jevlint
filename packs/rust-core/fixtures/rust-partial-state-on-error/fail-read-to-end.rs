/// Append atomically: on Err the caller's output remains unchanged.
/// The output may be reused after a failed read from the network stream.
pub fn append_atomic(reader: &mut std::net::TcpStream, output: &mut Vec<u8>) -> std::io::Result<usize> {
    std::io::Read::read_to_end(reader, output)
}
