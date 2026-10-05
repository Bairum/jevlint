/// Remote streams can continuously request outgoing datagrams.
/// The endpoint must bound queued datagrams independently of congestion control.
pub async fn produce(mut streams: tokio::sync::mpsc::UnboundedReceiver<[u8; 64]>, endpoint: tokio::sync::mpsc::UnboundedSender<[u8; 64]>) {
    while let Some(datagram) = streams.recv().await {
        if endpoint.send(datagram).is_err() { break; }
    }
}
