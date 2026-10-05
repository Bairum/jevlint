pub async fn record_and_yield(counter: &std::sync::Mutex<u64>) -> u64 {
    let value = {
        let mut guard = counter.lock().expect("counter poisoned");
        *guard += 1;
        *guard
    };
    tokio::task::yield_now().await;
    value
}
