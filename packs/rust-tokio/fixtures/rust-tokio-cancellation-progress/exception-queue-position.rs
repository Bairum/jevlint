/// Waits for capacity while leaving the record with its caller until admission.
pub async fn wait_for_capacity(capacity: &tokio::sync::Semaphore) -> bool {
    tokio::select! {
        permit = capacity.acquire() => permit.is_ok(),
        _ = tokio::time::sleep(std::time::Duration::from_millis(10)) => false,
    }
}
