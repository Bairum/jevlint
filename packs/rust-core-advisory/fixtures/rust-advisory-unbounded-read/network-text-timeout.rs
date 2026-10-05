use std::io::{self, Read};
use std::net::TcpStream;
use std::time::Duration;

pub fn receive_response(mut connection: TcpStream) -> io::Result<String> {
    connection.set_read_timeout(Some(Duration::from_secs(2)))?;
    let mut response = String::new();
    connection.read_to_string(&mut response)?;
    Ok(response)
}

