use std::io::{self, BufWriter, Write};
use std::net::TcpStream;

/// Diagnostic delivery is best effort and must not affect the main operation.
/// Only connection establishment and admission to the output buffer are required.
pub fn send_diagnostic(address: &str, message: &str) -> io::Result<()> {
    let mut writer = BufWriter::new(TcpStream::connect(address)?);
    writeln!(writer, "{message}")?;
    let _ = writer.flush();
    Ok(())
}
