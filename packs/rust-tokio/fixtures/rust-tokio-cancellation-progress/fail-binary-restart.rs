/// Reads the entire binary response through EOF on the current connection.
/// Status ticks must preserve all bytes belonging to that response.
pub async fn read_response(stream: &mut tokio::io::DuplexStream) -> std::io::Result<Vec<u8>> {
    loop {
        let mut response = Vec::new();
        tokio::select! {
            result = tokio::io::AsyncReadExt::read_to_end(stream, &mut response) => {
                result?;
                return Ok(response);
            }
            _ = tokio::time::sleep(std::time::Duration::from_millis(10)) => {}
        }
    }
}
