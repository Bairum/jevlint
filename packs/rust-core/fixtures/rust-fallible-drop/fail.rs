/// ReceiptWriter cleanup must never panic on routine peer disconnects, including
/// when dropped during another unwind. The buffer may contain pending bytes.
pub struct ReceiptWriter { pub writer: std::io::BufWriter<std::net::TcpStream> }
impl std::ops::Drop for ReceiptWriter {
    /// ReceiptWriter must not panic on ordinary peer disconnection.
    fn drop(&mut self) {
        std::io::Write::flush(&mut self.writer).expect("receipt cleanup failed");
    }
}
