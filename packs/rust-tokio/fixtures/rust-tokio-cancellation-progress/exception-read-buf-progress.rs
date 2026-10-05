/// Collects the entire binary response; housekeeping runs between receive attempts.
pub async fn collect_response(stream: &mut tokio::io::DuplexStream) -> std::io::Result<Vec<u8>> {
    let mut response = Vec::new();
    loop {
        tokio::select! {
            result = tokio::io::AsyncReadExt::read_buf(stream, &mut response) => {
                if result? == 0 {
                    return Ok(response);
                }
            }
            _ = tokio::time::sleep(std::time::Duration::from_millis(10)) => {}
        }
    }
}
