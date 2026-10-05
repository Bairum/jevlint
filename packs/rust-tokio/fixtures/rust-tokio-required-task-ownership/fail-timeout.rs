/// Returning releases the caller's exclusive use of this output slot.
/// No child may still be writing after either success or timeout.
pub async fn update_slot(slot: std::sync::Arc<tokio::sync::Mutex<Vec<u8>>>, deadline: std::time::Duration) -> std::io::Result<()> {
    let child = tokio::spawn(async move {
        tokio::time::sleep(std::time::Duration::from_secs(1)).await;
        slot.lock().await.push(7);
    });
    tokio::time::timeout(deadline, child).await
        .map_err(|_| std::io::Error::new(std::io::ErrorKind::TimedOut, "update deadline"))?
        .map_err(std::io::Error::other)?;
    Ok(())
}
