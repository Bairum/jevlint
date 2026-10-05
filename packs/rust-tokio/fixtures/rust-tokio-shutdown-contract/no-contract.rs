pub async fn collect() -> Vec<u8> {
    let (sender, mut receiver) = tokio::sync::mpsc::channel(1);
    sender.send(7_u8).await.expect("receiver alive");
    let task = tokio::spawn(async move {
        let mut values = Vec::new();
        while let Some(value) = receiver.recv().await { values.push(value); }
        values
    });
    let values = task.await.expect("task joined");
    drop(sender);
    values
}
