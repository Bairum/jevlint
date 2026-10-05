/// Finite best-effort work may finish after the initial wait budget.
pub async fn wait_for_finite_work(
    blocking_duration: std::time::Duration,
    wait_budget: std::time::Duration,
) -> u64 {
    let (started_sender, started_receiver) = tokio::sync::oneshot::channel::<()>();
    let mut worker = tokio::task::spawn_blocking(move || {
        started_sender.send(()).expect("coordinator is alive");
        std::thread::sleep(blocking_duration);
        7_u64
    });
    started_receiver.await.expect("worker started");
    match tokio::time::timeout(wait_budget, &mut worker).await {
        Ok(result) => result.expect("finite work did not panic"),
        Err(_elapsed) => worker.await.expect("late finite completion is allowed"),
    }
}
