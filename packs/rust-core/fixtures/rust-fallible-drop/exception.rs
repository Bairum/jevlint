/// TelemetryWriter cleanup is explicitly best effort. Pending messages are
/// disposable and routine peer disconnects need no error report or panic.
pub struct TelemetryWriter { pub writer: std::io::BufWriter<std::net::TcpStream> }
impl std::ops::Drop for TelemetryWriter {
    /// TelemetryWriter intentionally ignores disposable cleanup errors.
    fn drop(&mut self) {
        let _ = std::io::Write::flush(&mut self.writer);
    }
}
