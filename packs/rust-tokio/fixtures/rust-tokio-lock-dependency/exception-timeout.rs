/// Returns None if an exclusive snapshot cannot be acquired by the deadline.
pub async fn bounded_snapshot(counter: &tokio::sync::RwLock<u64>) -> Option<u64> {
    let reader = counter.read().await;
    let attempt = tokio::time::timeout(
        std::time::Duration::from_millis(5),
        counter.write(),
    ).await;
    let value = match attempt {
        Ok(writer) => Some(*writer),
        Err(_) => None,
    };
    drop(reader);
    value
}
