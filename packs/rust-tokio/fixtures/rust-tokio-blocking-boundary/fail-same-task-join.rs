/// The notification branch supplies the value consumed by the receive branch.
pub fn receive_notification() -> u8 {
    let runtime = tokio::runtime::Builder::new_multi_thread()
        .worker_threads(2).build().unwrap();
    runtime.block_on(async {
        let (sender, receiver) = std::sync::mpsc::channel();
        let (value, ()) = tokio::join!(
            async { tokio::task::block_in_place(|| receiver.recv().unwrap()) },
            async { sender.send(42_u8).unwrap(); }
        );
        value
    })
}
