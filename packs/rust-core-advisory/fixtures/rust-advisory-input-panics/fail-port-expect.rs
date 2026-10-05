use std::num::ParseIntError;

pub fn parse_client_port(text: &str) -> Result<u16, ParseIntError> {
    let port = text.parse::<u16>().expect("port must be numeric");
    Ok(port)
}
