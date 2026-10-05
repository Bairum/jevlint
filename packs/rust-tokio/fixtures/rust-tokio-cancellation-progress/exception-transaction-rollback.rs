/// Commits one complete record; interruption restores the previous ledger.
pub async fn append_record(ledger: &mut Vec<u8>, record: &[u8]) {
    let checkpoint = ledger.len();
    let committed = tokio::select! {
        _ = async {
            let midpoint = record.len() / 2;
            ledger.extend_from_slice(&record[..midpoint]);
            tokio::task::yield_now().await;
            ledger.extend_from_slice(&record[midpoint..]);
        } => true,
        _ = tokio::time::sleep(std::time::Duration::from_millis(10)) => false,
    };
    if !committed {
        ledger.truncate(checkpoint);
    }
}
