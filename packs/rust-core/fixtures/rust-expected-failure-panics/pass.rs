/// Parse a remote client's port; malformed requests are recoverable errors.
pub fn parse_client_port(text: &str) -> std::result::Result<u16, std::num::ParseIntError> {
    str::parse::<u16>(text)
}
