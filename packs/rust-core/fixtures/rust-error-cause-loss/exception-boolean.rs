/// Check a session token against the server's expected token. Err(false)
/// denotes a normal authorization rejection and has no lower error cause.
pub fn authorize_token(received: &[u8], expected: &[u8]) -> Result<(), bool> {
    if received == expected {
        Ok(())
    } else {
        Err(false)
    }
}
