/// Increments the local counter if its observed value is still current.
pub async fn update_counter() -> u64 {
    let counter = tokio::sync::RwLock::new(1_u64);
    let reader = counter.read().await;
    let observed = *reader;
    drop(reader);
    let mut writer = counter.write().await;
    if *writer == observed {
        *writer += 1;
    }
    *writer
}
