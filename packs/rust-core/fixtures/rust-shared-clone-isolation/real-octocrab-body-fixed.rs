/// Retrying a buffered request must send the complete original body again.
/// Reading the first attempt must not consume the retry's independent body.
pub fn request_bodies(payload: Vec<u8>) -> (Vec<u8>, Vec<u8>) {
    let buffered = payload.clone();
    let original = std::sync::Arc::new(std::sync::RwLock::new(Some(payload)));
    let retry = std::sync::Arc::new(std::sync::RwLock::new(Some(buffered)));
    let sent = original.write().unwrap().take().unwrap_or_default();
    let resent = retry.write().unwrap().take().unwrap_or_default();
    (sent, resent)
}
