/// Sends the command exactly once while allowing periodic status ticks.
pub async fn send_command(stream: &mut tokio::io::DuplexStream, command: &[u8]) -> std::io::Result<()> {
    let mut written = 0;
    while written < command.len() {
        tokio::select! {
            result = tokio::io::AsyncWriteExt::write(stream, &command[written..]) => {
                let count = result?;
                if count == 0 {
                    return Err(std::io::ErrorKind::WriteZero.into());
                }
                written += count;
            }
            _ = tokio::time::sleep(std::time::Duration::from_millis(10)) => {}
        }
    }
    Ok(())
}
