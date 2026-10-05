use std::io::{self, Read, Write};
use std::net::TcpStream;

pub fn relay(mut connection: TcpStream, destination: &mut impl Write) -> io::Result<u64> {
    const CHUNK_BYTES: u64 = 4096;
    let mut chunk = Vec::with_capacity(CHUNK_BYTES as usize);
    let mut transferred = 0_u64;
    loop {
        chunk.clear();
        Read::by_ref(&mut connection).take(CHUNK_BYTES).read_to_end(&mut chunk)?;
        if chunk.is_empty() {
            break;
        }
        destination.write_all(&chunk)?;
        transferred = transferred.saturating_add(chunk.len() as u64);
    }
    destination.flush()?;
    Ok(transferred)
}

