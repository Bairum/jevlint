/// A transparent owned text view until explicitly closed by its caller.
/// Dereferencing after `close` is programmer error and panics.
pub struct TextView {
    text: std::option::Option<std::string::String>,
}

impl TextView {
    pub fn new(text: std::string::String) -> TextView {
        TextView { text: Some(text) }
    }

    /// Releases the text. The caller must no longer dereference this view.
    pub fn close(&mut self) {
        self.text = None;
    }
}

impl std::ops::Deref for TextView {
    type Target = str;

    /// Transparently borrows the text without changing state while the view is open.
    /// Panics only on a view explicitly closed by the caller.
    fn deref(&self) -> &str {
        self.text.as_deref().expect("text view must remain open while accessed")
    }
}

pub fn text_length(text: std::string::String) -> usize {
    let view = TextView::new(text);
    let text: &str = &view;
    text.len()
}
