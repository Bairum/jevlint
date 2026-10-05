/// The synchronous request operation runs on a two-worker runtime.
pub fn handle_request() {
    let runtime = tokio::runtime::Builder::new_multi_thread()
        .worker_threads(2)
        .build()
        .expect("runtime construction");
    runtime.block_on(async {
        tokio::task::spawn(async {
            tokio::task::block_in_place(|| {
                std::thread::sleep(std::time::Duration::from_millis(100));
            });
        })
        .await
        .expect("request task completion");
    });
}
