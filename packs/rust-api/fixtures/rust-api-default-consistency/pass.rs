/// Runtime settings. Both `new()` and `Default` select the enabled baseline.
pub struct Settings {
    pub enabled: bool,
    pub retry_limit: u8,
}

impl Settings {
    /// Returns the same baseline settings as `Default`.
    pub fn new() -> Settings {
        <Settings as std::default::Default>::default()
    }
}

impl std::default::Default for Settings {
    fn default() -> Settings {
        Settings { enabled: true, retry_limit: 3 }
    }
}
