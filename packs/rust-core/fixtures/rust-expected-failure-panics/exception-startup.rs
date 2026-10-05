/// Application startup policy: an invalid administrator-supplied port is fatal.
/// This runs before serving requests; it deliberately panics rather than recover.
pub fn startup_port(setting: &str) -> u16 {
    str::parse::<u16>(setting).expect("fatal startup: invalid listen port")
}
