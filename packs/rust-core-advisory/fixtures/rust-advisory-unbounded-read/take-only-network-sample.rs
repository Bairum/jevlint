use std::io::{self, Read};
use std::net::TcpStream;

pub fn receive_sample(connection: TcpStream) -> io::Result<Vec<u8>> {
    let mut sample = Vec::new();
    connection.take(8192).read_to_end(&mut sample)?;
    Ok(sample)
}

