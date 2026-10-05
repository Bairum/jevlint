/// New and Default create an empty pending queue. Reserved capacity is not
/// part of the queue's application state.
pub struct PendingQueue {
    entries: Vec<String>,
}

impl PendingQueue {
    pub fn new() -> Self {
        Self { entries: Vec::with_capacity(64) }
    }

    pub fn len(&self) -> usize {
        self.entries.len()
    }
}

impl Default for PendingQueue {
    fn default() -> Self {
        Self { entries: Vec::new() }
    }
}
