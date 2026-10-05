/// Cleanup must observe completion of all pollers.
/// Pollers may be parked in a long-running provider operation.
pub async fn stop_pollers(resource: std::sync::Arc<()>) -> Result<(), tokio::task::JoinError> {
    let child = tokio::spawn(async move {
        let _resource = resource;
        std::future::pending::<()>().await;
    });
    let children = vec![child];
    for child in &children { child.abort(); }
    for child in children {
        match child.await {
            Err(error) if error.is_cancelled() => {},
            Err(error) => return Err(error),
            Ok(()) => {},
        }
    }
    Ok(())
}
