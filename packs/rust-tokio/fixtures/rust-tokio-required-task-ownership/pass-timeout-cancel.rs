/// No child may still write to the slot after this call returns.
pub async fn update_slot(slot: std::sync::Arc<tokio::sync::Mutex<Vec<u8>>>, deadline: std::time::Duration) -> std::io::Result<()> {
    let mut child = tokio::spawn(async move {
        tokio::time::sleep(std::time::Duration::from_secs(1)).await;
        slot.lock().await.push(7);
    });
    match tokio::time::timeout(deadline, &mut child).await {
        Ok(result) => result.map_err(std::io::Error::other),
        Err(_) => {
            child.abort();
            match child.await {
                Ok(()) => {},
                Err(error) if error.is_cancelled() => {},
                Err(error) => return Err(std::io::Error::other(error)),
            }
            Err(std::io::Error::new(std::io::ErrorKind::TimedOut, "update deadline"))
        }
    }
}
