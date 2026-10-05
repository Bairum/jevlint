/// Submit disposable telemetry. Success promises a locally accepted attempt,
/// not downstream delivery; late telemetry transport errors are ignored.
pub fn telemetry(stream: std::net::TcpStream) -> std::io::Result<()> {
    let mut writer = std::io::BufWriter::with_capacity(128, stream);
    std::io::Write::write_all(&mut writer, b"ping\n")?;
    Ok(())
}
