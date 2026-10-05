/// Writes an event. Callers inspect the I/O kind to distinguish a disconnected
/// destination from an invalid event supplied by the caller.
pub fn append_event(
    destination: &mut impl std::io::Write,
    event: &[u8],
) -> std::io::Result<()> {
    destination.write_all(event).map_err(|_| {
        std::io::Error::new(std::io::ErrorKind::InvalidInput, "event rejected")
    })
}
