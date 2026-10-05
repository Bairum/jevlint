/// ReceiptWriter reports required completion through finish. Peer disconnects
/// are ordinary errors; destructor fallback is explicitly nonpanicking.
pub struct ReceiptWriter { pub writer: std::io::BufWriter<std::net::TcpStream> }
impl ReceiptWriter {
    /// ReceiptWriter exposes required completion failures explicitly.
    pub fn finish(mut self) -> std::io::Result<()> {
        std::io::Write::flush(&mut self.writer)
    }
}
impl std::ops::Drop for ReceiptWriter {
    /// ReceiptWriter falls back to explicitly nonpanicking best-effort cleanup.
    fn drop(&mut self) {
        let _ = std::io::Write::flush(&mut self.writer);
    }
}
