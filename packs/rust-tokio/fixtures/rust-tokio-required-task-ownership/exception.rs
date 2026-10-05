/// Telemetry is best effort; loss during overflow or runtime shutdown is acceptable.
/// Request completion does not depend on the metric.
pub async fn record_telemetry(counter: std::sync::Arc<std::sync::atomic::AtomicUsize>) {
    let handle = tokio::task::spawn(async move {
        counter
            .try_update(
                std::sync::atomic::Ordering::Relaxed,
                std::sync::atomic::Ordering::Relaxed,
                |value| value.checked_add(1),
            )
            .map(|_| ())
    });
    std::mem::drop(handle);
}
