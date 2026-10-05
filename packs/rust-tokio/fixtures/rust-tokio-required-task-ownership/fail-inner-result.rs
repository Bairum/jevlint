/// A successful response requires the payload to reach the destination.
/// Transport failures must be returned to the caller.
pub async fn deliver(payload: Vec<u8>, destination: tokio::sync::mpsc::Sender<Vec<u8>>) -> std::io::Result<()> {
    let task = tokio::spawn(async move {
        destination.send(payload).await.map_err(|_| {
            std::io::Error::new(std::io::ErrorKind::BrokenPipe, "destination closed")
        })
    });
    let _ = task.await.map_err(std::io::Error::other)?;
    Ok(())
}
