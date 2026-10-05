/// Queue a receipt. Return the transport and pending bytes for the caller to
/// write and check before claiming delivery; this operation only prepares work.
pub fn prepare_receipt(stream: std::net::TcpStream) -> std::io::Result<(std::net::TcpStream, Vec<u8>)> {
    let mut writer = std::io::BufWriter::with_capacity(128, stream);
    std::io::Write::write_all(&mut writer, b"receipt\n")?;
    let (stream, pending) = writer.into_parts();
    let pending = pending.map_err(|_| std::io::Error::new(std::io::ErrorKind::Other, "writer panicked"))?;
    Ok((stream, pending))
}
