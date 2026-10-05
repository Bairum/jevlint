/// Receives a mandatory eight-byte frame on a persistent connection.
/// Housekeeping ticks must preserve every byte belonging to the current frame.
pub async fn receive_frame(mut stream: tokio::io::DuplexStream) -> std::io::Result<[u8; 8]> {
    let mut frame = [0_u8; 8];
    loop {
        tokio::select! {
            result = tokio::io::AsyncReadExt::read_exact(&mut stream, &mut frame) => {
                result?;
                return Ok(frame);
            }
            _ = tokio::time::sleep(std::time::Duration::from_millis(10)) => {}
        }
    }
}
