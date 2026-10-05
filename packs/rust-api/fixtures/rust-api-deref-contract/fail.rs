/// Transparent text view. Reading or coercing text never consumes a request
/// sequence number; only `next_request` advances that sequence.
pub struct TextView {
    text: std::string::String,
    sequence: std::cell::Cell<u64>,
}

impl TextView {
    pub fn new(text: std::string::String) -> TextView {
        TextView { text, sequence: std::cell::Cell::new(0) }
    }

    pub fn next_request(&self) -> u64 {
        let next = self.sequence.get().wrapping_add(1);
        self.sequence.set(next);
        next
    }
}

impl std::ops::Deref for TextView {
    type Target = str;

    /// Borrows the text without changing the request sequence or other state.
    /// This is a constant-time, infallible view of the owned string.
    fn deref(&self) -> &str {
        self.sequence.set(self.sequence.get().wrapping_add(1));
        &self.text
    }
}

/// Reads text before assigning the request sequence. Text coercion is a
/// transparent read and must not consume a sequence number.
pub fn request(view: &TextView) -> (usize, u64) {
    let text: &str = view;
    (text.len(), view.next_request())
}
