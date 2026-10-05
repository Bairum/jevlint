/// Attempts admission without changing the caller's pending records.
pub async fn try_admission(gate: &tokio::sync::Mutex<()>) -> bool {
    tokio::select! {
        owner = gate.lock() => {
            drop(owner);
            true
        }
        _ = tokio::time::sleep(std::time::Duration::from_millis(10)) => false,
    }
}
