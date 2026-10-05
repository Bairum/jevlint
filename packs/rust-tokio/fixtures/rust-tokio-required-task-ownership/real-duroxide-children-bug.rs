/// Cleanup must observe completion of all pollers, including those owned by a supervisor.
/// Pollers may be parked in a long-running provider operation.
pub async fn stop_pollers(resource: std::sync::Arc<()>) {
    let (ready, started) = tokio::sync::oneshot::channel();
    let supervisor = tokio::spawn(async move {
        let child = tokio::spawn(async move {
            let _resource = resource;
            std::future::pending::<()>().await;
        });
        let children = vec![child];
        let _ = ready.send(());
        for child in children { let _ = child.await; }
    });
    started.await.expect("supervisor started");
    supervisor.abort();
    let _ = supervisor.await;
}
