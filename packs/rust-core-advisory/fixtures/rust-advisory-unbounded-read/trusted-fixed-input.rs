use std::io::{self, Cursor, Read};

pub fn builtin_capabilities() -> io::Result<String> {
    const CAPABILITIES: &[u8] = b"version=1\nformat=binary\n";
    let mut input = Cursor::new(CAPABILITIES);
    let mut capabilities = String::new();
    input.read_to_string(&mut capabilities)?;
    Ok(capabilities)
}

