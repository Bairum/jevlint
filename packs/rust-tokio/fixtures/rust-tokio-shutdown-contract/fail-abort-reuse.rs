/// Shutdown finishes worker teardown before resetting its shared slot for reuse.
pub async fn shutdown_slot(slot: std::sync::Arc<tokio::sync::Mutex<Vec<u8>>>) {
    let worker_slot = slot.clone();
    let worker = tokio::spawn(async move {
        loop {
            tokio::task::yield_now().await;
            worker_slot.lock().await.push(1);
        }
    });
    worker.abort();
    slot.lock().await.clear();
}
