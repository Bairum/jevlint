pub async fn integrate_until_deadline(total: std::sync::Arc<std::sync::atomic::AtomicUsize>) {
    let worker_total = total.clone();
    let (started_sender, started_receiver) = tokio::sync::oneshot::channel();
    let (release_sender, release_receiver) = std::sync::mpsc::channel();
    let mut worker = tokio::task::spawn_blocking(move || {
        started_sender.send(()).unwrap();
        release_receiver.recv().unwrap();
        worker_total.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
    });
    started_receiver.await.unwrap();
    let result = tokio::time::timeout(std::time::Duration::ZERO, &mut worker).await;
    if result.is_err() {
        release_sender.send(()).unwrap();
        worker.await.unwrap();
    }
}
