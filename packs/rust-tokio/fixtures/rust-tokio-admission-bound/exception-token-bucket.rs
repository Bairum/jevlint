/// `tokens` meters requests per interval, not concurrent tasks.
/// A separately owned clock replenishes tokens each interval.
/// This serial consumer admits no child tasks; telemetry may be lost.
pub async fn meter(mut input: tokio::sync::mpsc::UnboundedReceiver<u64>, tokens: std::sync::Arc<tokio::sync::Semaphore>, destination: tokio::sync::mpsc::Sender<u64>) {
    while let Some(value) = input.recv().await {
        let Ok(token) = tokens.clone().try_acquire_owned() else { continue };
        token.forget();
        let _ = destination.try_send(value);
    }
}
