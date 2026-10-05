/// Temporary command-line prototype for comparing wire encodings.
/// This is a disposable local tool, not a request handler or library API.
pub fn prototype_port(text: &str) -> u16 {
    text.parse::<u16>().unwrap()
}
