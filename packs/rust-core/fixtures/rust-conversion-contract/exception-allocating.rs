/// Owned text with exactly the same Unicode scalar sequence as its source.
pub struct OwnedText(pub String);
impl std::convert::From<&str> for OwnedText {
    fn from(value: &str) -> OwnedText {
        OwnedText(String::from(value))
    }
}
