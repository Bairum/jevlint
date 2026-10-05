/// Success means the receipt reached the downstream writer, with all local
/// write and completion errors returned to the caller.
pub fn send_receipt(stream: std::net::TcpStream) -> std::io::Result<()> {
    let mut writer = std::io::BufWriter::with_capacity(128, stream);
    std::io::Write::write_all(&mut writer, b"receipt\n")?;
    let (stream, pending) = writer.into_parts();
    drop(pending);
    drop(stream);
    Ok(())
}
