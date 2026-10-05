use std::num::ParseIntError;

pub fn parse_client_port(text: &str) -> Result<u16, ParseIntError> {
    text.parse::<u16>()
}
