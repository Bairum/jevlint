/// The reference backend accepts calls from a current-thread runtime.
pub fn load_reference(reference: String) -> usize {
    let runtime = tokio::runtime::Builder::new_current_thread().build().unwrap();
    runtime.block_on(async move {
        tokio::task::spawn(async move {
            tokio::task::block_in_place(|| {
                tokio::runtime::Handle::current().block_on(async move {
                    reference.len()
                })
            })
        }).await.unwrap()
    })
}
