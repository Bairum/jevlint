use std::net::SocketAddr;
use std::num::ParseIntError;

pub fn parse_bind_address(port: &str) -> Result<SocketAddr, ParseIntError> {
    let port = port.parse::<u16>()?;
    let mut address: SocketAddr = "127.0.0.1:8000".parse().unwrap();
    address.set_port(port);
    Ok(address)
}
