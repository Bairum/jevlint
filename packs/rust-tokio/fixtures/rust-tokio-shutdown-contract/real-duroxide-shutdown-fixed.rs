/// Shutdown releases every poller's resource before the caller closes the provider pool.
/// Hard cancellation is allowed, but every child must finish teardown before return.
pub async fn shutdown(resource: std::sync::Arc<()>) -> Result<(), tokio::task::JoinError> {
    let poller = tokio::spawn(async move {
        let _resource = resource;
        std::future::pending::<()>().await;
    });
    let mut children = tokio::task::JoinSet::new();
    children.spawn(async move {
        poller.abort();
        match poller.await {
            Err(error) if error.is_cancelled() => Ok(()),
            result => result,
        }
    });
    while let Some(result) = children.join_next().await { result??; }
    Ok(())
}
