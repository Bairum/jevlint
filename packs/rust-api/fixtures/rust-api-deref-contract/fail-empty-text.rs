pub struct MessageText {
    text: String,
}

impl MessageText {
    /// Stores any text, including an empty message.
    pub fn new(text: String) -> Self {
        Self { text }
    }
}

impl std::ops::Deref for MessageText {
    type Target = str;

    /// Transparently borrows the stored text, including empty messages.
    /// Borrowing never changes state and is infallible for every constructed value.
    fn deref(&self) -> &str {
        assert!(!self.text.is_empty(), "message must contain text");
        &self.text
    }
}

pub fn empty_message_length() -> usize {
    let message = MessageText::new(String::new());
    let text: &str = &message;
    text.len()
}
