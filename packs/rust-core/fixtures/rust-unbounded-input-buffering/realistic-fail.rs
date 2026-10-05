const REQUEST_LIMIT: usize = 4096;

#[derive(Clone, Copy)]
pub enum Method {
    Status,
    Echo,
}

pub struct RequestHeader<'a> {
    pub method: Method,
    pub route: &'a str,
}

pub struct Request<'a> {
    pub header: RequestHeader<'a>,
    pub body: &'a [u8],
}

#[derive(Clone, Copy)]
pub enum Status {
    Ok,
    BadRequest,
    TooLarge,
}

impl Status {
    pub fn wire_header(self) -> &'static [u8] {
        match self {
            Self::Ok => b"200 OK\n",
            Self::BadRequest => b"400 BAD REQUEST\n",
            Self::TooLarge => b"413 REQUEST TOO LARGE\n",
        }
    }

    pub fn for_input_error(error: &std::io::Error) -> Self {
        match error.kind() {
            std::io::ErrorKind::InvalidData => Self::TooLarge,
            _ => Self::BadRequest,
        }
    }
}

/// Read one complete EOF-framed request from an untrusted remote peer.
/// Enforce the 4096-byte limit before input-driven buffer growth can exceed
/// the limit plus one detection byte. Reject oversized frames, never accept
/// a truncated prefix; every accepted frame must have reached peer EOF.
pub fn read_request(stream: &mut std::net::TcpStream) -> std::io::Result<Vec<u8>> {
    let mut bytes = Vec::new();
    std::io::Read::read_to_end(stream, &mut bytes)?;
    if bytes.len() > REQUEST_LIMIT {
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidData,
            "request exceeds 4096 bytes",
        ));
    }
    Ok(bytes)
}

/// Parse the ASCII command and route without copying the borrowed header.
pub fn parse_header(line: &str) -> std::io::Result<RequestHeader<'_>> {
    let (command, route) = line.split_once(' ').ok_or_else(|| {
        std::io::Error::new(std::io::ErrorKind::InvalidInput, "missing route")
    })?;
    let method = match (command, route) {
        ("STATUS", "/health") => Method::Status,
        ("ECHO", "/echo") => Method::Echo,
        _ => {
            return Err(std::io::Error::new(
                std::io::ErrorKind::InvalidInput,
                "unknown command or route",
            ));
        }
    };
    Ok(RequestHeader { method, route })
}

/// Decode only a complete frame returned by read_request; borrow its body.
pub fn decode_request(frame: &[u8]) -> std::io::Result<Request<'_>> {
    let end = frame.iter().position(|byte| *byte == b'\n').ok_or_else(|| {
        std::io::Error::new(std::io::ErrorKind::InvalidInput, "incomplete header")
    })?;
    let line = std::str::from_utf8(&frame[..end]).map_err(|_| {
        std::io::Error::new(std::io::ErrorKind::InvalidInput, "invalid header encoding")
    })?;
    let header = parse_header(line)?;
    let body = &frame[end + 1..];
    if matches!(header.method, Method::Status) && !body.is_empty() {
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "status request must have an empty body",
        ));
    }
    Ok(Request { header, body })
}

/// Respond once, then mark EOF on the response half of the connection.
pub fn write_response(
    stream: &mut std::net::TcpStream,
    status: Status,
    body: &[u8],
) -> std::io::Result<()> {
    std::io::Write::write_all(stream, status.wire_header())?;
    std::io::Write::write_all(stream, body)?;
    stream.shutdown(std::net::Shutdown::Write)
}

/// Handle one remote request; transport failures are returned to the caller.
pub fn serve_connection(stream: &mut std::net::TcpStream) -> std::io::Result<()> {
    let frame = match read_request(stream) {
        Ok(frame) => frame,
        Err(error) if error.kind() == std::io::ErrorKind::InvalidData => {
            return write_response(stream, Status::for_input_error(&error), b"frame rejected\n");
        }
        Err(error) => return Err(error),
    };
    let request = match decode_request(&frame) {
        Ok(request) => request,
        Err(error) => {
            return write_response(stream, Status::for_input_error(&error), b"invalid request\n");
        }
    };
    let body = match request.header.method {
        Method::Status => b"ready\n".as_slice(),
        Method::Echo => request.body,
    };
    write_response(stream, Status::Ok, body)
}
