pub async fn read_request_file(path: std::path::PathBuf) -> std::io::Result<Vec<u8>> {
    let (sender, receiver) = tokio::sync::oneshot::channel();
    std::thread::spawn(move || {
        let _ = sender.send(std::fs::read(path));
    });
    receiver.await.expect("file reader completed")
}
