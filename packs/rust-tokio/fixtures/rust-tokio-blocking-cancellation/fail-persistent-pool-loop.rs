/// Shutdown must stop the session worker and release its blocking-pool slot.
/// Session updates may not occur after this function returns.
pub async fn shutdown_session(total: std::sync::Arc<std::sync::atomic::AtomicUsize>) {
    let (started_sender, started_receiver) = tokio::sync::oneshot::channel();
    let worker = tokio::task::spawn_blocking(move || {
        started_sender.send(()).unwrap();
        loop {
            total.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
            std::thread::sleep(std::time::Duration::from_millis(10));
        }
    });
    started_receiver.await.unwrap();
    worker.abort();
}
