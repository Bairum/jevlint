#[derive(Debug)]
pub struct ConnectionFailure {
    pub endpoint: std::net::SocketAddr,
    pub diagnostic: String,
    pub cause: Option<std::io::Error>,
}

/// Connect to the service. Failures retain an inspectable cause so callers
/// distinguish ConnectionRefused from TimedOut without parsing diagnostic text.
pub fn connect(endpoint: std::net::SocketAddr) -> Result<std::net::TcpStream, ConnectionFailure> {
    std::net::TcpStream::connect(endpoint).map_err(|cause| ConnectionFailure {
        endpoint,
        diagnostic: cause.to_string(),
        cause: None,
    })
}
