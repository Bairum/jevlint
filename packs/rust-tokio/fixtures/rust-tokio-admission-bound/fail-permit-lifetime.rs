/// Producers submit external requests continuously.
/// The semaphore limits the number of live response buffers until delivery finishes.
pub async fn respond(mut requests: tokio::sync::mpsc::UnboundedReceiver<u64>, limit: std::sync::Arc<tokio::sync::Semaphore>, destination: tokio::sync::mpsc::Sender<Vec<u8>>) {
    while let Some(value) = requests.recv().await {
        let permit = limit.clone().acquire_owned().await.expect("limit open");
        let destination = destination.clone();
        tokio::spawn(async move {
            let response = value.to_le_bytes().to_vec();
            drop(permit);
            let _ = destination.send(response).await;
        });
    }
}
