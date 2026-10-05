/// Result of checking a block's inherents. New and Default both start with
/// a successful result and no recorded errors.
pub struct CheckInherentsResult {
    pub okay: bool,
    pub fatal_error: bool,
    pub errors: Vec<String>,
}

impl CheckInherentsResult {
    pub fn new() -> Self {
        Self { okay: true, fatal_error: false, errors: Vec::new() }
    }

    pub fn record_error(&mut self, error: String, fatal: bool) {
        self.okay = false;
        self.fatal_error |= fatal;
        self.errors.push(error);
    }
}

impl Default for CheckInherentsResult {
    fn default() -> Self {
        Self { okay: false, fatal_error: false, errors: Vec::new() }
    }
}
