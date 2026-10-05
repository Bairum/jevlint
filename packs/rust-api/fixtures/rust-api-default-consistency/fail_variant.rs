/// New and Default both select the online service baseline.
pub enum ServiceMode {
    Online { concurrency: usize },
    Offline,
}

impl ServiceMode {
    pub fn new() -> Self {
        Self::Online { concurrency: 4 }
    }
}

impl Default for ServiceMode {
    fn default() -> Self {
        Self::Offline
    }
}
