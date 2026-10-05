/// New and Default both select the local audit-log baseline.
pub struct AuditOptions {
    pub output: Option<String>,
    pub retain_days: u16,
}

impl AuditOptions {
    pub fn new() -> Self {
        Self { output: Some(String::from("audit.log")), retain_days: 30 }
    }
}

impl Default for AuditOptions {
    fn default() -> Self {
        Self { output: None, retain_days: 30 }
    }
}
