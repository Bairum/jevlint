/// Shutdown returns every accepted record after the worker reaches EOF.
/// The channel and all its producers belong to this coordinator.
pub async fn drain_shutdown() -> std::vec::Vec<u8> {
    let (sender, mut receiver) = tokio::sync::mpsc::channel::<u8>(1);
    let retained_sender = sender.clone();
    sender.send(7).await.expect("receiver is alive");
    let worker = tokio::spawn(async move {
        let mut drained = std::vec::Vec::new();
        while let Some(record) = receiver.recv().await {
            drained.push(record);
        }
        drained
    });
    drop(sender);
    drop(retained_sender);
    worker.await.expect("worker did not panic")
}
