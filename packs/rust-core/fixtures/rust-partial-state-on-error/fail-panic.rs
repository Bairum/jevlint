/// Replace the payload atomically. If the builder panics, return its panic
/// payload and preserve state for subsequent reads and replacement attempts.
pub fn rebuild(state: &mut Vec<u8>, build: impl FnOnce() -> Vec<u8>) -> Result<(), Box<dyn std::any::Any + Send>> {
    state.clear();
    let replacement = std::panic::catch_unwind(std::panic::AssertUnwindSafe(build))?;
    *state = replacement;
    Ok(())
}
