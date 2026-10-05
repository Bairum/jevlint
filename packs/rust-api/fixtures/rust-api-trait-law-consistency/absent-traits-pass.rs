pub struct Payload {
    pub bytes: std::vec::Vec<u8>,
}

impl Payload {
    pub fn byte_len(&self) -> usize {
        self.bytes.len()
    }
}

impl std::hash::Hash for Payload {
    fn hash<H: std::hash::Hasher>(&self, state: &mut H) {
        std::hash::Hash::hash(&self.bytes, state);
    }
}
