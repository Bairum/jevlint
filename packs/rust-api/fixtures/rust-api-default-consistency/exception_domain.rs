/// A listener owns a caller-selected socket; there is no default address.
pub struct Listener {
    socket: std::net::TcpListener,
}

impl Listener {
    pub fn bind(address: std::net::SocketAddr) -> std::io::Result<Self> {
        let socket = std::net::TcpListener::bind(address)?;
        Ok(Self { socket })
    }

    pub fn local_address(&self) -> std::io::Result<std::net::SocketAddr> {
        self.socket.local_addr()
    }
}
