pub struct TextView {
    text: String,
    sequence: std::cell::Cell<u64>,
}

impl TextView {
    pub fn new(text: String) -> Self {
        Self { text, sequence: std::cell::Cell::new(0) }
    }

    pub fn next_request(&self) -> u64 {
        let next = self.sequence.get().wrapping_add(1);
        self.sequence.set(next);
        next
    }
}

impl std::ops::Deref for TextView {
    type Target = str;

    fn deref(&self) -> &str {
        self.sequence.set(self.sequence.get().wrapping_add(1));
        &self.text
    }
}

pub fn request(view: &TextView) -> (usize, u64) {
    let text: &str = view;
    (text.len(), view.next_request())
}
