/// A receipt journal buffered over a collector connection.
/// Required delivery is reported by `finish`; destructor cleanup is best effort
/// and must not panic when the collector disconnects or during unwinding.
pub struct ReceiptJournal {
    writer: std::io::BufWriter<std::net::TcpStream>,
    accepted_receipts: u64,
    finished: bool,
}

/// A receipt ready for the collector's tab-separated journal format.
pub struct Receipt {
    pub order_id: u64,
    pub amount_cents: u64,
    pub customer: String,
}

/// Local acceptance and buffering state, not a delivery acknowledgement.
pub struct JournalStatus {
    pub accepted_receipts: u64,
    pub queued_bytes: usize,
}

impl ReceiptJournal {
    /// Connects to a collector and bounds how long socket writes may wait.
    /// Connection and socket configuration errors are returned to the caller.
    pub fn connect(address: std::net::SocketAddr) -> std::io::Result<Self> {
        let stream = std::net::TcpStream::connect_timeout(
            &address,
            std::time::Duration::from_secs(2),
        )?;
        Self::from_stream(stream)
    }

    /// Takes ownership of a connected socket and configures bounded writes.
    /// No receipt is sent until `append` is called.
    pub fn from_stream(stream: std::net::TcpStream) -> std::io::Result<Self> {
        stream.set_write_timeout(Some(std::time::Duration::from_secs(2)))?;
        Ok(Self {
            writer: std::io::BufWriter::with_capacity(16 * 1024, stream),
            accepted_receipts: 0,
            finished: false,
        })
    }

    /// Appends a complete receipt, rejecting field separators before writing.
    /// Acceptance is local, not proof of delivery. After a write error the
    /// caller must discard this journal because a partial record may exist.
    pub fn append(&mut self, receipt: &Receipt) -> std::io::Result<()> {
        if receipt.customer.contains(['\t', '\r', '\n']) {
            return Err(std::io::Error::new(
                std::io::ErrorKind::InvalidInput,
                "customer must occupy one journal field",
            ));
        }
        std::io::Write::write_fmt(
            &mut self.writer,
            format_args!(
                "{}\t{}\t{}\n",
                receipt.order_id, receipt.amount_cents, receipt.customer,
            ),
        )?;
        self.accepted_receipts += 1;
        Ok(())
    }

    /// Returns local counters and bytes still buffered without performing I/O.
    /// An empty buffer does not imply that the collector persisted receipts.
    pub fn status(&self) -> JournalStatus {
        JournalStatus {
            accepted_receipts: self.accepted_receipts,
            queued_bytes: self.writer.buffer().len(),
        }
    }

    /// Flushes accepted receipts and reports disconnection or other I/O errors.
    /// Success means the socket accepted all bytes, not remote durability.
    /// This consumes the journal so no receipts can be appended afterward.
    pub fn finish(mut self) -> std::io::Result<JournalStatus> {
        std::io::Write::flush(&mut self.writer)?;
        self.finished = true;
        Ok(self.status())
    }
}

impl std::ops::Drop for ReceiptJournal {
    /// Attempts best-effort cleanup only when explicit finish did not succeed.
    /// Collector disconnection is an ordinary cleanup failure; this must not
    /// panic, including while another failure is already unwinding the stack.
    fn drop(&mut self) {
        if !self.finished {
            let _ = std::io::Write::flush(&mut self.writer);
        }
    }
}
