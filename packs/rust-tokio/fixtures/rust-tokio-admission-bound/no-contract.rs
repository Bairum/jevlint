pub async fn dispatch(mut input: tokio::sync::mpsc::UnboundedReceiver<u64>, semaphore: std::sync::Arc<tokio::sync::Semaphore>) {
    while let Some(value) = input.recv().await {
        let semaphore = semaphore.clone();
        tokio::spawn(async move {
            let _permit = semaphore.acquire_owned().await;
            tokio::task::yield_now().await;
            value.wrapping_add(1)
        });
    }
}
