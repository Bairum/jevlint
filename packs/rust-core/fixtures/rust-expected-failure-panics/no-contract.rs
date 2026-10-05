pub fn parse_client_port(text: &str) -> std::result::Result<u16, std::num::ParseIntError> {
    let port: u16 = str::parse::<u16>(text).expect("client port should be valid");
    Ok(port)
}
