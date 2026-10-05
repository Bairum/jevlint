pub async fn compare_snapshots() -> bool {
    let data = tokio::sync::RwLock::new(vec![1_u64, 2]);
    let first = data.read().await;
    let second = data.read().await;
    *first == *second
}
