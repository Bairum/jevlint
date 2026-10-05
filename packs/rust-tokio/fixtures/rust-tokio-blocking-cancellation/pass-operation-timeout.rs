/// The worker stops when its receive budget expires; completion precedes return.
pub async fn receive_with_budget(budget: std::time::Duration) -> bool {
    let (sender, receiver) = std::sync::mpsc::channel::<u8>();
    let (started_sender, started_receiver) = tokio::sync::oneshot::channel();
    let worker = tokio::task::spawn_blocking(move || {
        started_sender.send(()).unwrap();
        receiver.recv_timeout(budget).is_ok()
    });
    started_receiver.await.unwrap();
    let received = worker.await.unwrap();
    drop(sender);
    received
}
