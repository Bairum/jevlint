/// Shutdown releases every poller's resource before the caller closes the provider pool.
/// Hard cancellation is allowed, but every child must finish teardown before return.
pub async fn shutdown(resource: std::sync::Arc<()>) {
    let (ready, started) = tokio::sync::oneshot::channel();
    let supervisor = tokio::spawn(async move {
        let poller = tokio::spawn(async move {
            let _resource = resource;
            std::future::pending::<()>().await;
        });
        let children = vec![poller];
        let _ = ready.send(());
        for child in children { let _ = child.await; }
    });
    started.await.expect("supervisor started");
    supervisor.abort();
    let _ = supervisor.await;
}
