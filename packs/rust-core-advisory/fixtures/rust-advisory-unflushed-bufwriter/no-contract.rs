pub fn send_receipt(stream: std::net::TcpStream) -> std::io::Result<()> {
    let mut writer = std::io::BufWriter::with_capacity(128, stream);
    std::io::Write::write_all(&mut writer, b"receipt\n")?;
    Ok(())
}
