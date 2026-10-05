/// Append streaming bytes. On an I/O error the retained prefix is available
/// to the caller for progress reporting and subsequent processing.
pub fn append_progress(reader: &mut std::net::TcpStream, output: &mut Vec<u8>) -> std::io::Result<usize> {
    std::io::Read::read_to_end(reader, output)
}
