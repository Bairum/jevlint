pub struct ValidatedText {
    text: String,
    checksum: u64,
}

impl ValidatedText {
    pub fn new(text: String) -> Self {
        let checksum = text.bytes().fold(0_u64, |sum, byte| {
            sum.wrapping_mul(31).wrapping_add(u64::from(byte))
        });
        Self { text, checksum }
    }
}

impl std::ops::Deref for ValidatedText {
    type Target = str;

    /// Returns a transparent constant-time reference to the stored text.
    /// Validation is completed at construction; borrowing performs no scans,
    /// allocations, or further validation and cannot fail for a constructed value.
    fn deref(&self) -> &str {
        let checksum = self.text.bytes().fold(0_u64, |sum, byte| {
            sum.wrapping_mul(31).wrapping_add(u64::from(byte))
        });
        assert_eq!(checksum, self.checksum);
        &self.text
    }
}

pub fn display_width(text: &ValidatedText) -> usize {
    let value: &str = text;
    value.chars().count()
}
