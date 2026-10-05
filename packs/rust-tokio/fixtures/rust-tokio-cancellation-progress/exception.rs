/// Receives one frame, or rejects and closes the connection on deadline expiry.
/// A rejected connection's incomplete frame is not required.
pub async fn receive_frame(mut stream: tokio::io::DuplexStream) -> std::io::Result<[u8; 8]> {
    let mut frame = [0_u8; 8];
    let outcome = tokio::time::timeout(
        std::time::Duration::from_millis(100),
        tokio::io::AsyncReadExt::read_exact(&mut stream, &mut frame),
    )
    .await;
    match outcome {
        Ok(result) => {
            result?;
            Ok(frame)
        }
        Err(_) => {
            std::mem::drop(stream);
            Err(std::io::Error::new(
                std::io::ErrorKind::TimedOut,
                "incomplete frame; connection discarded",
            ))
        }
    }
}
