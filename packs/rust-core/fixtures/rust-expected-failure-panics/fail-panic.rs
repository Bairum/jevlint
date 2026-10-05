use std::io::{self, Read};

/// Read one request byte from a client connection.
/// Disconnects and transport failures must be returned to the caller.
pub fn read_request_byte(reader: &mut impl Read) -> io::Result<u8> {
    let mut byte = [0];
    match reader.read_exact(&mut byte) {
        Ok(()) => Ok(byte[0]),
        Err(error) => panic!("client read failed: {error}"),
    }
}
