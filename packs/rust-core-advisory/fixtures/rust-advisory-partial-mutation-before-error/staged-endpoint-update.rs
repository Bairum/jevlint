use std::num::ParseIntError;

pub struct Endpoint {
    pub name: String,
    pub port: u16,
}

pub fn update_endpoint(
    endpoint: &mut Endpoint,
    name: &str,
    port: &str,
) -> Result<(), ParseIntError> {
    let candidate = Endpoint {
        name: name.to_owned(),
        port: port.parse()?,
    };
    *endpoint = candidate;
    Ok(())
}
