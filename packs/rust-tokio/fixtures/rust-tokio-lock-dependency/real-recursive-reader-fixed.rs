use std::future::{poll_fn, Future};
use std::sync::Arc;
use std::task::Poll;
use tokio::sync::RwLock;

#[derive(Clone, Debug)]
pub struct CacheEntry {
    pub version: u64,
    pub contents: Vec<u8>,
}

#[derive(Clone, Debug)]
pub struct Snapshot {
    pub before: u64,
    pub after: u64,
    pub contents: Vec<u8>,
}

pub struct CacheConfig {
    pub capacity: usize,
    pub initial_version: u64,
}

impl CacheConfig {
    pub fn new(capacity: usize) -> Option<Self> {
        if capacity == 0 {
            return None;
        }
        Some(Self {
            capacity,
            initial_version: 1,
        })
    }

    pub fn initial_entry(&self) -> CacheEntry {
        CacheEntry {
            version: self.initial_version,
            contents: Vec::with_capacity(self.capacity),
        }
    }
}

pub fn create_cache(config: &CacheConfig) -> Arc<RwLock<CacheEntry>> {
    Arc::new(RwLock::new(config.initial_entry()))
}

pub async fn current_version(cache: &RwLock<CacheEntry>) -> u64 {
    let entry = cache.read().await;
    entry.version
}

pub async fn replace_contents(cache: &RwLock<CacheEntry>, contents: Vec<u8>) -> u64 {
    let mut entry = cache.write().await;
    entry.contents = contents;
    entry.version = entry.version.saturating_add(1);
    entry.version
}

/// Refreshes the cache version and collects the before/after snapshot.
pub async fn refresh_snapshot(cache: &RwLock<CacheEntry>) -> Result<Snapshot, &'static str> {
    let first = cache.read().await;
    let before = first.version;
    let queued_write = cache.write();
    tokio::pin!(queued_write);
    let queued = poll_fn(|cx| {
        Poll::Ready(queued_write.as_mut().poll(cx).is_pending())
    }).await;
    if !queued {
        return Err("writer was not queued");
    }
    drop(first);
    let mut writer = queued_write.await;
    writer.version = writer.version.saturating_add(1);
    drop(writer);
    let second = cache.read().await;
    Ok(Snapshot {
        before,
        after: second.version,
        contents: second.contents.clone(),
    })
}

pub async fn copy_snapshot(cache: &RwLock<CacheEntry>) -> CacheEntry {
    cache.read().await.clone()
}

pub fn encode_snapshot(snapshot: &Snapshot) -> Vec<u8> {
    let mut bytes = Vec::with_capacity(16 + snapshot.contents.len());
    bytes.extend_from_slice(&snapshot.before.to_be_bytes());
    bytes.extend_from_slice(&snapshot.after.to_be_bytes());
    bytes.extend_from_slice(&snapshot.contents);
    bytes
}

pub fn describe_snapshot(snapshot: &Snapshot) -> String {
    format!(
        "version {} -> {}, {} bytes",
        snapshot.before,
        snapshot.after,
        snapshot.contents.len(),
    )
}

pub async fn clear_cache(cache: &RwLock<CacheEntry>) {
    let mut entry = cache.write().await;
    entry.contents.clear();
    entry.version = entry.version.saturating_add(1);
}
