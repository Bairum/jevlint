/// Cache identity is namespace plus an ASCII-case-insensitive name.
/// Any UTF-8 name is accepted; no normalization occurs on construction.
pub struct CacheKey {
    pub namespace: u16,
    pub name: std::string::String,
}

impl CacheKey {
    pub fn new(namespace: u16, name: std::string::String) -> CacheKey {
        CacheKey { namespace, name }
    }
}

impl std::cmp::PartialEq for CacheKey {
    fn eq(&self, other: &Self) -> bool {
        self.namespace == other.namespace && self.name.eq_ignore_ascii_case(&other.name)
    }
}

impl std::cmp::Eq for CacheKey {}

impl std::hash::Hash for CacheKey {
    fn hash<H: std::hash::Hasher>(&self, state: &mut H) {
        std::hash::Hash::hash(&self.namespace, state);
        std::hash::Hash::hash(&self.name.to_ascii_lowercase(), state);
    }
}
