/// External telemetry is disposable; requests beyond concurrent capacity are discarded.
/// No lossless delivery promise applies.
pub async fn record(mut input: tokio::sync::mpsc::UnboundedReceiver<u64>, capacity: std::sync::Arc<tokio::sync::Semaphore>, destination: tokio::sync::mpsc::Sender<u64>) {
    while let Some(value) = input.recv().await {
        let Ok(permit) = capacity.clone().try_acquire_owned() else { continue };
        let destination = destination.clone();
        tokio::spawn(async move {
            let _ = destination.send(value).await;
            drop(permit);
        });
    }
}
