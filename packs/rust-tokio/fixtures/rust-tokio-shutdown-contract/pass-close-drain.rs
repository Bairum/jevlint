/// Shutdown rejects new sends and returns every accepted record.
/// All senders belong to this coordinator; no outstanding send permits exist.
pub async fn shutdown_records() -> Vec<u64> {
    let (sender, mut receiver) = tokio::sync::mpsc::channel(4);
    sender.send(1_u64).await.expect("receiver alive");
    sender.send(2_u64).await.expect("receiver alive");
    receiver.close();
    let mut records = Vec::new();
    while let Some(record) = receiver.recv().await { records.push(record); }
    drop(sender);
    records
}
