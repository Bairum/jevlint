/// Upstream admission supplies one finite batch.
/// The local producer closes before processing starts.
/// `work_limit` applies only to executing work.
pub async fn process_admitted_jobs(
    admitted_jobs: std::vec::Vec<u64>,
    work_limit: std::num::NonZeroUsize,
) {
    let (sender, mut receiver) = tokio::sync::mpsc::unbounded_channel::<u64>();
    for job in admitted_jobs {
        sender.send(job).expect("receiver is alive");
    }
    drop(sender);
    let permits = std::sync::Arc::new(tokio::sync::Semaphore::new(work_limit.get()));
    let mut tasks = tokio::task::JoinSet::new();
    while let Some(job) = receiver.recv().await {
        let permits = std::sync::Arc::clone(&permits);
        tasks.spawn(async move {
            let _permit = permits.acquire_owned().await.expect("semaphore remains open");
            tokio::task::yield_now().await;
            job.wrapping_add(1)
        });
    }
    while let Some(result) = tasks.join_next().await {
        result.expect("job did not panic");
    }
}
