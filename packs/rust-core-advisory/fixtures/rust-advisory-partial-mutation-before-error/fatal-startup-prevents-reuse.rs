use std::num::ParseIntError;

pub struct Startup {
    pub service: String,
    pub port: u16,
}

/// Startup configuration belongs to this executable. A conversion failure
/// terminates the process; callers may use the state only after success.
pub fn configure_or_exit(
    startup: &mut Startup,
    service: &str,
    port: &str,
) -> Result<(), ParseIntError> {
    startup.service = service.to_owned();
    let conversion = port.parse::<u16>();
    if let Err(error) = &conversion {
        eprintln!("invalid startup port: {error}");
        std::process::exit(2);
    }
    startup.port = conversion?;
    Ok(())
}
