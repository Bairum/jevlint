/// Success means the receipt reached the downstream TCP stream. Return local
/// writing errors and the unbuffered stream for subsequent requests.
/// This does not guarantee peer acknowledgment or crash durability.
pub fn send_receipt(stream: std::net::TcpStream) -> std::io::Result<std::net::TcpStream> {
    let mut writer = std::io::BufWriter::with_capacity(128, stream);
    std::io::Write::write_all(&mut writer, b"receipt\n")?;
    writer.into_inner().map_err(|error| error.into_error())
}
