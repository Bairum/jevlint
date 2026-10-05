/// Queue a receipt and return ownership. The caller must complete the returned
/// writer and handle completion errors before reporting the receipt as sent.
pub fn queue_receipt(stream: std::net::TcpStream) -> std::io::Result<std::io::BufWriter<std::net::TcpStream>> {
    let mut writer = std::io::BufWriter::with_capacity(128, stream);
    std::io::Write::write_all(&mut writer, b"receipt\n")?;
    Ok(writer)
}
