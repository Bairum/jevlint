pub struct CachedText {
    path: std::path::PathBuf,
    text: String,
    revision: u64,
}

impl CachedText {
    pub fn open(path: std::path::PathBuf) -> std::io::Result<Self> {
        let text = std::fs::read_to_string(&path)?;
        Ok(Self { path, text, revision: 0 })
    }

    /// Reads the backing file and replaces the cached text on success.
    /// Each successful refresh advances the revision.
    pub fn refresh(&mut self) -> std::io::Result<u64> {
        let text = std::fs::read_to_string(&self.path)?;
        self.text = text;
        self.revision = self.revision.wrapping_add(1);
        Ok(self.revision)
    }

    pub fn revision(&self) -> u64 {
        self.revision
    }
}

impl std::ops::Deref for CachedText {
    type Target = str;

    /// Transparently borrows cached text in constant time without I/O,
    /// state changes, or failure. Use `refresh` to reload the backing file.
    fn deref(&self) -> &str {
        &self.text
    }
}

pub fn summary(text: &CachedText) -> (usize, u64) {
    let contents: &str = text;
    (contents.len(), text.revision())
}
