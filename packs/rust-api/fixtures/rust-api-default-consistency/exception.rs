/// Explicit settings may select any retry policy; Default disables retries.
pub struct Settings {
    pub retry_limit: u8,
}

impl Settings {
    /// Selects the caller's policy, not the default policy.
    pub fn new(retry_limit: u8) -> Settings {
        Settings { retry_limit }
    }
}

impl std::default::Default for Settings {
    fn default() -> Settings {
        Settings { retry_limit: 0 }
    }
}
