pub struct CachedDocument {
    pub title: String,
    pub pages: Vec<String>,
}

/// Release an in-memory document. Its owned strings and vectors manage their
/// own storage; no external operation or completion obligation is involved.
pub fn release(document: CachedDocument) {
    std::mem::drop(document);
}
