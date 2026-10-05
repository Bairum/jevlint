/// External producers can submit jobs for the lifetime of the service.
/// `task_limit` bounds live job tasks, including tasks waiting to start work.
/// Job results are disposable.
pub async fn admit_jobs(
    mut jobs: tokio::sync::mpsc::UnboundedReceiver<u64>,
    task_limit: std::num::NonZeroUsize,
) {
    let permits = std::sync::Arc::new(tokio::sync::Semaphore::new(task_limit.get()));
    let mut tasks = tokio::task::JoinSet::new();
    while let Some(job) = jobs.recv().await {
        let permits = std::sync::Arc::clone(&permits);
        tasks.spawn(async move {
            let permit = permits.acquire_owned().await.expect("semaphore remains open");
            tokio::task::yield_now().await;
            let result = job.wrapping_add(1);
            drop(permit);
            result
        });
        while let Some(result) = tasks.try_join_next() {
            result.expect("job did not panic");
        }
    }
    while let Some(result) = tasks.join_next().await {
        result.expect("job did not panic");
    }
}
