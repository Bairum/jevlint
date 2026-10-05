/// The reference backend accepts calls from a current-thread runtime.
pub fn load_reference(reference: String) -> usize {
    let runtime = tokio::runtime::Builder::new_current_thread().build().unwrap();
    runtime.block_on(async move {
        tokio::task::spawn_blocking(move || {
            let private_runtime = tokio::runtime::Builder::new_current_thread()
                .build().unwrap();
            private_runtime.block_on(async move { reference.len() })
        }).await.unwrap()
    })
}
