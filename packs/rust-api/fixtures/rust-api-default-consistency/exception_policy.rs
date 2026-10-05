/// Default is a disabled configuration for deserialization.
/// New selects the interactive application's enabled startup policy.
pub struct Settings {
    pub enabled: bool,
    pub retry_limit: u8,
}

impl Settings {
    pub fn new() -> Self {
        Self { enabled: true, retry_limit: 3 }
    }
}

impl Default for Settings {
    fn default() -> Self {
        Self { enabled: false, retry_limit: 0 }
    }
}
