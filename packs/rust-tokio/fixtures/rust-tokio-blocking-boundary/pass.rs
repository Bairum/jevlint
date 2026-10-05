/// Runs a request task on an explicitly current-thread Tokio runtime.
/// Its delay must leave the runtime free to poll other request tasks and timers.
pub fn handle_request() {
    let runtime = tokio::runtime::Builder::new_current_thread()
        .enable_time()
        .build()
        .expect("runtime construction");
    runtime.block_on(async {
        tokio::task::spawn(async {
            tokio::time::sleep(std::time::Duration::from_millis(100)).await;
        })
        .await
        .expect("request task completion");
    });
}
