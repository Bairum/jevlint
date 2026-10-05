/// Write the receipt downstream and report local write and completion errors.
/// Success does not promise crash durability or a peer acknowledgment.
pub fn send_receipt(stream: std::net::TcpStream) -> std::io::Result<()> {
    let mut writer = std::io::BufWriter::with_capacity(128, stream);
    std::io::Write::write_all(&mut writer, b"receipt\n")?;
    std::io::Write::flush(&mut writer)?;
    drop(writer);
    Ok(())
}
