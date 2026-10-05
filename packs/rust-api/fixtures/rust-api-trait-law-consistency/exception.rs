/// Cache identity is namespace plus exact name. Hashing groups namespaces
/// together to keep the wire-independent hash input limited to the name.
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
        self.namespace == other.namespace && self.name == other.name
    }
}

impl std::cmp::Eq for CacheKey {}

impl std::hash::Hash for CacheKey {
    fn hash<H: std::hash::Hasher>(&self, state: &mut H) {
        std::hash::Hash::hash(&self.name, state);
    }
}
