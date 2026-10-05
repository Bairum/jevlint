/// Success means the receipt was written to the downstream TCP stream.
/// Return all local write and completion errors. No peer acknowledgment or
/// crash-durability guarantee is provided.
pub fn send_receipt(stream: std::net::TcpStream) -> std::io::Result<()> {
    let mut writer = std::io::BufWriter::with_capacity(128, stream);
    std::io::Write::write_all(&mut writer, b"receipt\n")?;
    std::io::Write::flush(&mut writer)?;
    Ok(())
}
