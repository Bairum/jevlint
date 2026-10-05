/// Hard shutdown permits loss of disposable records.
/// Worker teardown must finish before return.
pub async fn hard_shutdown() {
    let (sender, mut receiver) = tokio::sync::mpsc::channel::<u8>(1);
    sender.send(7).await.expect("receiver is alive");
    let worker = tokio::spawn(async move {
        while let Some(_record) = receiver.recv().await {
            tokio::task::yield_now().await;
        }
    });
    worker.abort();
    drop(sender);
    match worker.await {
        Ok(()) => {}
        Err(error) => assert!(error.is_cancelled(), "worker panicked: {error}"),
    }
}
