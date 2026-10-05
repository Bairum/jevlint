/// Increments the local counter and returns the committed value.
pub async fn update_counter() -> u64 {
    let counter = tokio::sync::RwLock::new(1_u64);
    let reader = counter.read().await;
    let mut writer = counter.write().await;
    *writer = *reader + 1;
    *writer
}
