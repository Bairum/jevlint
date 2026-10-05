pub enum Delivery {
    Complete(std::io::Result<()>),
    Pending(tokio::task::JoinHandle<std::io::Result<()>>),
}

/// Deadline expiry returns ownership to the caller; a pending delivery remains required.
pub async fn deliver(value: u64, destination: tokio::sync::mpsc::Sender<u64>, deadline: std::time::Duration) -> Delivery {
    let mut child = tokio::spawn(async move {
        destination.send(value).await.map_err(|_| {
            std::io::Error::new(std::io::ErrorKind::BrokenPipe, "destination closed")
        })
    });
    match tokio::time::timeout(deadline, &mut child).await {
        Ok(result) => Delivery::Complete(result.map_err(std::io::Error::other).and_then(|inner| inner)),
        Err(_) => Delivery::Pending(child),
    }
}
