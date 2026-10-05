/// Success sends an approved receipt downstream and reports transport errors.
/// Unapproved receipts return PermissionDenied and may abandon queued output.
pub fn send_receipt(stream: std::net::TcpStream, approved: bool) -> std::io::Result<()> {
    let mut writer = std::io::BufWriter::with_capacity(128, stream);
    std::io::Write::write_all(&mut writer, b"receipt\n")?;
    if !approved {
        let (_stream, _pending) = writer.into_parts();
        return Err(std::io::Error::new(std::io::ErrorKind::PermissionDenied, "receipt not approved"));
    }
    std::io::Write::flush(&mut writer)?;
    Ok(())
}
