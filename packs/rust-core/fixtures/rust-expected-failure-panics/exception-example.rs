use std::net::SocketAddr;

/// Documentation example showing how to specify a loopback listener address.
/// The address is a literal supplied by the example itself.
pub fn example_listener_address() -> SocketAddr {
    "127.0.0.1:8080".parse().expect("the example uses a valid address")
}
