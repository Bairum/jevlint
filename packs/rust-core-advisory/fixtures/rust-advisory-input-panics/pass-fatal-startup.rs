use std::io;
use std::net::TcpListener;

/// Initializes the application listener before any requests are accepted.
///
/// Deployment policy treats an absent or malformed SERVICE_PORT as fatal.
/// Configuration panics terminate startup; socket binding failures are returned
/// so the supervisor can classify operating-system resource failures.
pub fn initialize_service() -> Result<TcpListener, io::Error> {
    let port = std::env::var("SERVICE_PORT").expect("SERVICE_PORT is required at startup");
    let port = port.parse::<u16>().expect("SERVICE_PORT must be an integer port");
    TcpListener::bind(("127.0.0.1", port))
}
