/// Processes one trusted finite batch with no live producer.
/// `work_limit` limits concurrent execution; the batch size bounds task count.
pub async fn process_finite_batch(
    jobs: std::vec::Vec<u64>,
    work_limit: std::num::NonZeroUsize,
) {
    let permits = std::sync::Arc::new(tokio::sync::Semaphore::new(work_limit.get()));
    let mut tasks = tokio::task::JoinSet::new();
    for job in jobs {
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
