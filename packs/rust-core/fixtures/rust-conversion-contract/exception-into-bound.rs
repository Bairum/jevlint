/// Retains the exact byte sequence supplied by the caller.
pub struct Payload(pub Vec<u8>);
impl std::convert::From<Vec<u8>> for Payload {
    fn from(value: Vec<u8>) -> Payload {
        Payload(value)
    }
}

pub fn payload_from<T: std::convert::Into<Vec<u8>>>(value: T) -> Payload {
    Payload(value.into())
}
