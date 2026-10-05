pub async fn try_reenter(counter: &tokio::sync::Mutex<u64>) -> bool {
    let mut owner = counter.lock().await;
    match counter.try_lock() {
        Ok(another) => drop(another),
        Err(_) => *owner += 1,
    }
    true
}
