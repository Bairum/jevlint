/// Drain within the grace period, then discard disposable work.
/// Task teardown must finish even when the deadline expires.
pub async fn shutdown_with_deadline(mut records: tokio::sync::mpsc::Receiver<u64>, grace: std::time::Duration) -> Result<(), tokio::task::JoinError> {
    records.close();
    let mut worker = tokio::spawn(async move {
        while let Some(_record) = records.recv().await {
            tokio::task::yield_now().await;
        }
    });
    match tokio::time::timeout(grace, &mut worker).await {
        Ok(result) => result,
        Err(_) => {
            worker.abort();
            match worker.await {
                Err(error) if error.is_cancelled() => Ok(()),
                result => result,
            }
        }
    }
}
