/// Copies the source value to the destination.
pub async fn copy_between_locks() -> u64 {
    let source = tokio::sync::RwLock::new(1_u64);
    let destination = tokio::sync::RwLock::new(0_u64);
    let reader = source.read().await;
    let mut writer = destination.write().await;
    *writer = *reader;
    *writer
}
