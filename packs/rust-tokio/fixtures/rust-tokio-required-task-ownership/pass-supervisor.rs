/// Acknowledges admission, not delivery. The returned supervisor owns all delivery results.
pub fn enqueue(value: u64, destination: tokio::sync::mpsc::Sender<u64>, supervisor: &mut tokio::task::JoinSet<std::io::Result<()>>) {
    supervisor.spawn(async move {
        destination.send(value).await.map_err(|_| {
            std::io::Error::new(std::io::ErrorKind::BrokenPipe, "destination closed")
        })
    });
}

/// Completion requires delivery of every admitted value; errors reach the caller.
pub async fn complete(supervisor: &mut tokio::task::JoinSet<std::io::Result<()>>) -> std::io::Result<()> {
    let mut failure = None;
    while let Some(result) = supervisor.join_next().await {
        if let Err(error) = result.map_err(std::io::Error::other).and_then(|inner| inner) {
            failure.get_or_insert(error);
        }
    }
    match failure { Some(error) => Err(error), None => Ok(()) }
}
