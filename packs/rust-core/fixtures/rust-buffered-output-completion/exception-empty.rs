/// Prepare and release an unused local buffer. This operation submits no
/// output and promises no transport operation or delivery acknowledgment.
pub fn release_empty(stream: std::net::TcpStream) -> std::io::Result<()> {
    let writer = std::io::BufWriter::with_capacity(128, stream);
    drop(writer);
    Ok(())
}
