/// Begins shutdown; the returned supervisor owns teardown completion.
/// The caller must await each task before reusing its resources.
pub fn begin_shutdown(mut shutdown: tokio::sync::oneshot::Receiver<()>) -> tokio::task::JoinSet<()> {
    let mut workers = tokio::task::JoinSet::new();
    workers.spawn(async move {
        let _ = (&mut shutdown).await;
        tokio::task::yield_now().await;
    });
    workers
}

/// Returns only after all supervised teardown completes.
pub async fn finish_shutdown(mut workers: tokio::task::JoinSet<()>) -> Result<(), tokio::task::JoinError> {
    let mut failure = None;
    while let Some(result) = workers.join_next().await {
        if let Err(error) = result { failure.get_or_insert(error); }
    }
    match failure { Some(error) => Err(error), None => Ok(()) }
}
