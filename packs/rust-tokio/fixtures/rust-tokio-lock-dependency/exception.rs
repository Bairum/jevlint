/// Serializes a complete message write on the protected connection.
pub async fn serialized_write() -> std::io::Result<[u8; 2]> {
    let (writer, mut reader) = tokio::io::duplex(1);
    let connection = tokio::sync::Mutex::new(writer);
    let peer = tokio::spawn(async move {
        let mut bytes = [0_u8; 2];
        tokio::io::AsyncReadExt::read_exact(&mut reader, &mut bytes).await?;
        Ok::<[u8; 2], std::io::Error>(bytes)
    });
    let mut guard = connection.lock().await;
    let write_result = tokio::io::AsyncWriteExt::write_all(&mut *guard, b"ok").await;
    drop(guard);
    drop(connection);
    let received = peer.await.expect("peer did not panic");
    write_result?;
    received
}
