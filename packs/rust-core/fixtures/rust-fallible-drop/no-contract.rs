pub struct ReceiptWriter { pub writer: std::io::BufWriter<std::net::TcpStream> }
impl std::ops::Drop for ReceiptWriter {
    fn drop(&mut self) {
        std::io::Write::flush(&mut self.writer).expect("receipt cleanup failed");
    }
}
