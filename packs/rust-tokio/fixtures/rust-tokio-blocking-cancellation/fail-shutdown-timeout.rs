/// Once runtime shutdown returns, the completed-generation counter must remain unchanged.
pub fn shutdown_generation() -> usize {
    let runtime = tokio::runtime::Builder::new_multi_thread().build().unwrap();
    let total = std::sync::Arc::new(std::sync::atomic::AtomicUsize::new(0));
    let worker_total = total.clone();
    let (started_sender, started_receiver) = tokio::sync::oneshot::channel();
    let (release_sender, release_receiver) = std::sync::mpsc::channel();
    let worker = runtime.spawn_blocking(move || {
        started_sender.send(()).unwrap();
        release_receiver.recv().unwrap();
        worker_total.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
    });
    runtime.block_on(started_receiver).unwrap();
    runtime.shutdown_timeout(std::time::Duration::from_millis(1));
    release_sender.send(()).unwrap();
    let observer = tokio::runtime::Builder::new_current_thread().build().unwrap();
    observer.block_on(worker).unwrap();
    total.load(std::sync::atomic::Ordering::SeqCst)
}
