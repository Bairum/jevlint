/// Reads the complete UTF-8 response through EOF on the current connection.
/// Status ticks must preserve all response text already consumed.
pub async fn read_response(stream: &mut tokio::io::DuplexStream) -> std::io::Result<String> {
    let mut response = String::new();
    loop {
        tokio::select! {
            result = tokio::io::AsyncReadExt::read_to_string(stream, &mut response) => {
                result?;
                return Ok(response);
            }
            _ = tokio::time::sleep(std::time::Duration::from_millis(10)) => {}
        }
    }
}
