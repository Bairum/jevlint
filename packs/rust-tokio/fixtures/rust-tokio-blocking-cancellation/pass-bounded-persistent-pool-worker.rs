/// Exactly one blocking worker belongs to this session.
/// Closing its command channel requests stop; shutdown observes completion.
pub async fn run_session(commands: Vec<u8>) -> usize {
    let (sender, receiver) = std::sync::mpsc::channel::<u8>();
    let (started_sender, started_receiver) = tokio::sync::oneshot::channel();
    let worker = tokio::task::spawn_blocking(move || {
        started_sender.send(()).unwrap();
        let mut count = 0;
        while let Ok(_command) = receiver.recv() {
            count += 1;
        }
        count
    });
    started_receiver.await.unwrap();
    for command in commands {
        sender.send(command).unwrap();
    }
    drop(sender);
    worker.await.unwrap()
}
