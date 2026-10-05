/// Trusted, fixed, locally generated data, not an untrusted frame boundary.
pub fn read_builtin() -> std::io::Result<Vec<u8>> {
    let mut input = std::io::Cursor::new(b"built-in defaults".as_slice());
    let mut bytes = Vec::new();
    std::io::Read::read_to_end(&mut input, &mut bytes)?;
    Ok(bytes)
}
