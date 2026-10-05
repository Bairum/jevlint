pub async fn try_update(counter: &tokio::sync::RwLock<u64>) -> Option<u64> {
    let observed = counter.read().await;
    let outcome = match counter.try_write() {
        Ok(mut writer) => {
            *writer += 1;
            Some(*writer)
        }
        Err(_) => None,
    };
    drop(observed);
    outcome
}
