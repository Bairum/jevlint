/// Byte content is the entire value; spare allocation capacity has no meaning.
pub struct Bytes(pub Box<[u8]>);
impl std::convert::From<Vec<u8>> for Bytes {
    fn from(value: Vec<u8>) -> Bytes {
        Bytes(Vec::into_boxed_slice(value))
    }
}
