/// Records are required during service operation; shutdown may discard unsent records.
pub async fn send_until_shutdown(
    queue: &tokio::sync::mpsc::Sender<String>,
    record: String,
    shutdown: &mut tokio::sync::watch::Receiver<bool>,
) -> bool {
    tokio::select! {
        result = queue.send(record) => result.is_ok(),
        _ = shutdown.wait_for(|stopping| *stopping) => false,
    }
}
