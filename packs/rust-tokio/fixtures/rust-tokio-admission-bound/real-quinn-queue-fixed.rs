pub struct Datagram {
    pub bytes: [u8; 64],
    pub reservation: tokio::sync::OwnedSemaphorePermit,
}

/// Remote streams can continuously request outgoing datagrams.
/// Queue reservations cover datagrams until the endpoint finishes transmission.
pub async fn produce(mut streams: tokio::sync::mpsc::UnboundedReceiver<[u8; 64]>, endpoint: tokio::sync::mpsc::UnboundedSender<Datagram>, capacity: std::sync::Arc<tokio::sync::Semaphore>) {
    while let Some(bytes) = streams.recv().await {
        let reservation = capacity.clone().acquire_owned().await.expect("capacity open");
        if endpoint.send(Datagram { bytes, reservation }).is_err() { break; }
    }
}

pub async fn transmit(mut endpoint: tokio::sync::mpsc::UnboundedReceiver<Datagram>, wire: tokio::sync::mpsc::Sender<[u8; 64]>) {
    while let Some(datagram) = endpoint.recv().await {
        let _ = wire.send(datagram.bytes).await;
        drop(datagram.reservation);
    }
}
