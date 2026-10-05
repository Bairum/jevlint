#[derive(Debug, PartialEq, Eq)]
pub enum DecodeError {
    TooLarge,
    Encoding,
    MissingHeader,
    InvalidSequence,
    UnknownCommand,
    InvalidResource,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Command {
    Fetch,
    Store,
}

#[derive(Debug, PartialEq, Eq)]
pub struct Request<'a> {
    pub sequence: u64,
    pub command: Command,
    pub resource: &'a str,
    pub body: &'a str,
}

#[derive(Debug, Clone, Copy)]
pub struct Limits {
    pub packet_bytes: usize,
    pub resource_bytes: usize,
}

impl Default for Limits {
    fn default() -> Self {
        Self {
            packet_bytes: 4096,
            resource_bytes: 128,
        }
    }
}

impl Limits {
    pub fn check_packet(&self, bytes: &[u8]) -> Result<(), DecodeError> {
        if bytes.len() > self.packet_bytes {
            return Err(DecodeError::TooLarge);
        }
        Ok(())
    }

    pub fn check_resource(&self, resource: &str) -> Result<(), DecodeError> {
        if resource.is_empty() || resource.len() > self.resource_bytes {
            return Err(DecodeError::InvalidResource);
        }
        if !resource.bytes().all(|byte| byte.is_ascii_alphanumeric() || byte == b'-') {
            return Err(DecodeError::InvalidResource);
        }
        Ok(())
    }
}

/// Decode a packet received from a remote client.
/// All malformed packets, including non-UTF-8 text, return DecodeError;
/// the connection owner can reject one packet and continue serving others.
pub fn decode_request(bytes: &[u8], limits: Limits) -> Result<Request<'_>, DecodeError> {
    limits.check_packet(bytes)?;
    let text = std::str::from_utf8(bytes).map_err(|_| DecodeError::Encoding)?;
    let (header, body) = text.split_once('\n').ok_or(DecodeError::MissingHeader)?;
    let mut fields = header.splitn(3, ' ');
    let command = parse_command(fields.next().ok_or(DecodeError::MissingHeader)?)?;
    let sequence = parse_sequence(fields.next().ok_or(DecodeError::MissingHeader)?)?;
    let resource = fields.next().ok_or(DecodeError::MissingHeader)?;
    limits.check_resource(resource)?;
    Ok(Request {
        sequence,
        command,
        resource,
        body,
    })
}

pub fn parse_command(text: &str) -> Result<Command, DecodeError> {
    match text {
        "FETCH" => Ok(Command::Fetch),
        "STORE" => Ok(Command::Store),
        _ => Err(DecodeError::UnknownCommand),
    }
}

pub fn parse_sequence(text: &str) -> Result<u64, DecodeError> {
    text.parse().map_err(|_| DecodeError::InvalidSequence)
}

#[derive(Debug, PartialEq, Eq)]
pub struct Reply {
    pub sequence: u64,
    pub accepted_bytes: usize,
}

pub fn acknowledge(request: &Request<'_>) -> Reply {
    let accepted_bytes = match request.command {
        Command::Fetch => 0,
        Command::Store => request.body.len(),
    };
    Reply {
        sequence: request.sequence,
        accepted_bytes,
    }
}

pub fn handle_packet(bytes: &[u8], limits: Limits) -> Result<Reply, DecodeError> {
    let request = decode_request(bytes, limits)?;
    Ok(acknowledge(&request))
}
