/// Receives one complete frame while preserving its bytes across status ticks.
pub async fn receive_frame(stream: &mut tokio::io::DuplexStream) -> std::io::Result<[u8; 8]> {
    let mut frame = [0_u8; 8];
    let mut filled = 0;
    while filled < frame.len() {
        tokio::select! {
            result = tokio::io::AsyncReadExt::read(stream, &mut frame[filled..]) => {
                let count = result?;
                if count == 0 {
                    return Err(std::io::ErrorKind::UnexpectedEof.into());
                }
                filled += count;
            }
            _ = tokio::time::sleep(std::time::Duration::from_millis(10)) => {}
        }
    }
    Ok(frame)
}
