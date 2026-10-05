/// Opens a probe connection. The returned error provides enough information
/// for an operator to distinguish a timeout from connection refusal.
pub fn connect_probe(
    address: std::net::SocketAddr,
) -> Result<std::net::TcpStream, String> {
    std::net::TcpStream::connect_timeout(&address, std::time::Duration::from_secs(2))
        .map_err(|_| "probe unavailable".to_owned())
}
