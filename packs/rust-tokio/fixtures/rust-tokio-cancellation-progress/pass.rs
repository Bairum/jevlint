/// Receives a mandatory eight-byte frame on a persistent connection.
/// Housekeeping ticks must preserve every byte belonging to the current frame.
pub async fn receive_frame(mut stream: tokio::io::DuplexStream) -> std::io::Result<[u8; 8]> {
    let mut frame = [0_u8; 8];
    {
        let read = tokio::io::AsyncReadExt::read_exact(&mut stream, &mut frame);
        tokio::pin!(read);
        loop {
            tokio::select! {
                result = &mut read => {
                    result?;
                    break;
                }
                _ = tokio::time::sleep(std::time::Duration::from_millis(10)) => {}
            }
        }
    }
    Ok(frame)
}
