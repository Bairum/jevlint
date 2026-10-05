/// Each request owns its counter; no other task accesses this mutex.
pub fn handle_request() {
    let runtime = tokio::runtime::Builder::new_current_thread()
        .enable_time()
        .build()
        .expect("runtime construction");
    runtime.block_on(async {
        tokio::task::spawn(async {
            let counter = std::sync::Mutex::new(0_u32);
            {
                let mut guard = counter.lock().expect("private counter is not poisoned");
                *guard += 1;
            }
            tokio::time::sleep(std::time::Duration::from_millis(100)).await;
        })
        .await
        .expect("request task completion");
    });
}
