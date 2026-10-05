/// A single dedicated worker processes commands until its owner closes the channel.
/// Shutdown returns after every accepted command and the worker have completed.
pub async fn process_commands(commands: Vec<u8>) -> usize {
    let (sender, receiver) = std::sync::mpsc::channel::<u8>();
    let (started_sender, started_receiver) = tokio::sync::oneshot::channel();
    let worker = std::thread::spawn(move || {
        started_sender.send(()).unwrap();
        let mut processed = 0;
        while let Ok(_command) = receiver.recv() {
            processed += 1;
        }
        processed
    });
    started_receiver.await.unwrap();
    for command in commands {
        sender.send(command).unwrap();
    }
    drop(sender);
    tokio::task::spawn_blocking(move || worker.join().unwrap()).await.unwrap()
}
