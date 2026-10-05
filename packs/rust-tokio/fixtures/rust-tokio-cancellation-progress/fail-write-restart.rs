/// Sends the command exactly once on this persistent ordered byte stream.
/// Housekeeping ticks may delay completion but may not change the command bytes.
pub async fn send_command(
    stream: &mut tokio::io::DuplexStream,
    command: &[u8],
) -> std::io::Result<()> {
    loop {
        tokio::select! {
            result = tokio::io::AsyncWriteExt::write_all(stream, command) => return result,
            _ = tokio::time::sleep(std::time::Duration::from_millis(10)) => {}
        }
    }
}
