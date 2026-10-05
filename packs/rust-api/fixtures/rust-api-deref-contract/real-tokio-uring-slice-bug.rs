/// An owned view of a buffer's initialized bytes within its reserved extent.
pub struct BufferSlice {
    buffer: Vec<u8>,
    begin: usize,
    end: usize,
}

impl BufferSlice {
    /// Creates a view spanning the allocation's full writable capacity.
    /// Empty and partially initialized buffers are supported.
    pub fn full(buffer: Vec<u8>) -> Self {
        let end = buffer.capacity();
        Self { buffer, begin: 0, end }
    }

    pub fn reserved_len(&self) -> usize {
        self.end - self.begin
    }
}

impl std::ops::Deref for BufferSlice {
    type Target = [u8];

    /// Transparently borrows the initialized bytes within this view.
    /// Borrowing is constant-time and infallible for views created by `full`,
    /// including buffers whose initialized length is below their capacity.
    fn deref(&self) -> &[u8] {
        &self.buffer[self.begin..self.end]
    }
}

pub fn prepare_read_buffer() -> (usize, usize) {
    let view = BufferSlice::full(Vec::with_capacity(32));
    let initialized: &[u8] = &view;
    (initialized.len(), view.reserved_len())
}
