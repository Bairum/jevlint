#[derive(Debug)]
pub struct HeaderError;

pub fn parse_packet_sequence(bytes: &[u8]) -> Result<u32, HeaderError> {
    let prefix = bytes.get(..4).ok_or(HeaderError)?;
    let word: [u8; 4] = prefix.try_into().expect("prefix is exactly four bytes");
    Ok(u32::from_be_bytes(word))
}
