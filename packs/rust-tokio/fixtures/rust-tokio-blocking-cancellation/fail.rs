/// No worker may append after the shared log is cleared for its next generation.
pub async fn recycle_log() -> std::vec::Vec<u8> {
    let log = std::sync::Arc::new(std::sync::Mutex::new(std::vec![9_u8]));
    let worker_log = std::sync::Arc::clone(&log);
    let (command_sender, command_receiver) = std::sync::mpsc::channel::<bool>();
    let (started_sender, started_receiver) = tokio::sync::oneshot::channel::<()>();
    let worker = tokio::task::spawn_blocking(move || {
        started_sender.send(()).expect("coordinator is alive");
        if command_receiver.recv().expect("coordinator sends a command") {
            worker_log.lock().expect("log is not poisoned").push(7);
        }
    });
    started_receiver.await.expect("worker started");
    worker.abort();
    log.lock().expect("log is not poisoned").clear();
    command_sender.send(true).expect("command receiver is alive");
    worker.await.expect("worker completion");
    std::sync::Arc::try_unwrap(log)
        .expect("completed worker released log")
        .into_inner()
        .expect("log is not poisoned")
}
