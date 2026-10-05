/// Producers submit external requests continuously.
/// The semaphore limits live response buffers through delivery.
/// Destination queue capacity is managed separately; delivery is best effort.
pub async fn respond(mut requests: tokio::sync::mpsc::UnboundedReceiver<u64>, limit: std::sync::Arc<tokio::sync::Semaphore>, destination: tokio::sync::mpsc::Sender<Vec<u8>>) {
    while let Some(value) = requests.recv().await {
        let permit = limit.clone().acquire_owned().await.expect("limit open");
        let destination = destination.clone();
        tokio::spawn(async move {
            let response = value.to_le_bytes().to_vec();
            let _ = destination.send(response).await;
            drop(permit);
        });
    }
}
