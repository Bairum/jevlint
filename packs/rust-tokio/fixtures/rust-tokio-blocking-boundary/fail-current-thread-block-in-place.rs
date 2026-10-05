/// Request execution uses a current-thread runtime.
pub fn read_request_file(path: std::path::PathBuf) -> std::io::Result<Vec<u8>> {
    let runtime = tokio::runtime::Builder::new_current_thread().build().unwrap();
    runtime.block_on(async {
        tokio::task::spawn(async move {
            tokio::task::block_in_place(|| std::fs::read(path))
        }).await.unwrap()
    })
}
