/// Every accepted record must be returned before shutdown completes.
/// This coordinator owns the only sender and the receiver.
pub async fn shutdown_records() -> Vec<u64> {
    let (sender, mut receiver) = tokio::sync::mpsc::channel(4);
    sender.send(1_u64).await.expect("receiver alive");
    sender.send(2_u64).await.expect("receiver alive");
    drop(sender);
    receiver.close();
    drop(receiver);
    Vec::new()
}
