/// Telemetry is best effort; a saturated queue may discard the sample.
pub async fn emit_sample(queue: &tokio::sync::mpsc::Sender<String>, sample: String) -> bool {
    tokio::select! {
        result = queue.send(sample) => result.is_ok(),
        _ = tokio::time::sleep(std::time::Duration::from_millis(10)) => false,
    }
}
