use std::collections::HashMap;
use std::fmt;
use std::hash::{Hash, Hasher};

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub struct ResourceId {
    pub namespace: u32,
    pub slot: u64,
}

/// Generation and label describe the origin of a request for a resource.
/// Resource identity is determined by its namespace and slot.
#[derive(Clone, Debug)]
pub struct ResourceKey {
    pub id: ResourceId,
    pub generation: u64,
    pub label: String,
}

impl PartialEq for ResourceKey {
    fn eq(&self, other: &Self) -> bool {
        self.id == other.id
    }
}

impl Eq for ResourceKey {}

impl Hash for ResourceKey {
    fn hash<H: Hasher>(&self, state: &mut H) {
        self.id.hash(state);
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum CachePolicy {
    KeepExisting,
    ReplaceExisting,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub struct CacheConfig {
    pub max_entries: usize,
    pub policy: CachePolicy,
}

#[derive(Debug, PartialEq, Eq)]
pub enum CacheError {
    ZeroCapacity,
    Full { limit: usize },
    Occupied(ResourceId),
}

impl fmt::Display for CacheError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::ZeroCapacity => write!(formatter, "cache capacity must be positive"),
            Self::Full { limit } => write!(formatter, "cache capacity of {limit} is exhausted"),
            Self::Occupied(id) => write!(formatter, "resource {}:{} is cached", id.namespace, id.slot),
        }
    }
}

impl std::error::Error for CacheError {}

#[derive(Debug, PartialEq, Eq, Hash)]
pub struct Resource {
    pub bytes: Vec<u8>,
    pub content_type: String,
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Hash)]
pub struct CacheStats {
    pub hits: u64,
    pub writes: u64,
    pub removals: u64,
}

/// An entry-count-bounded cache partitioned into application namespaces.
pub struct ResourceCache {
    config: CacheConfig,
    entries: HashMap<ResourceKey, Resource>,
    stats: CacheStats,
}

impl ResourceCache {
    pub fn new(config: CacheConfig) -> Result<Self, CacheError> {
        if config.max_entries == 0 {
            return Err(CacheError::ZeroCapacity);
        }
        Ok(Self {
            config,
            entries: HashMap::new(),
            stats: CacheStats::default(),
        })
    }

    /// Returns the previous resource when the replacement policy permits it.
    pub fn insert(
        &mut self,
        key: ResourceKey,
        resource: Resource,
    ) -> Result<Option<Resource>, CacheError> {
        let occupied = self.entries.contains_key(&key);
        if occupied && self.config.policy == CachePolicy::KeepExisting {
            return Err(CacheError::Occupied(key.id));
        }
        if !occupied && self.entries.len() >= self.config.max_entries {
            return Err(CacheError::Full { limit: self.config.max_entries });
        }
        let previous = self.entries.insert(key, resource);
        self.stats.writes += 1;
        Ok(previous)
    }

    pub fn get(&mut self, key: &ResourceKey) -> Option<&Resource> {
        let resource = self.entries.get(key);
        if resource.is_some() {
            self.stats.hits += 1;
        }
        resource
    }

    pub fn remove(&mut self, key: &ResourceKey) -> Option<Resource> {
        let resource = self.entries.remove(key);
        if resource.is_some() {
            self.stats.removals += 1;
        }
        resource
    }

    /// Removes all resources belonging to a namespace and returns their count.
    pub fn invalidate_namespace(&mut self, namespace: u32) -> usize {
        let before = self.entries.len();
        self.entries.retain(|key, _| key.id.namespace != namespace);
        let removed = before - self.entries.len();
        self.stats.removals += removed as u64;
        removed
    }

    pub fn len(&self) -> usize {
        self.entries.len()
    }

    pub fn is_empty(&self) -> bool {
        self.entries.is_empty()
    }

    pub fn stats(&self) -> CacheStats {
        self.stats
    }
}
