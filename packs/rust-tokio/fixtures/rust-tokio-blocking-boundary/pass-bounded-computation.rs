pub async fn checksum_header(header: [u8; 16]) -> u16 {
    tokio::task::spawn(async move {
        header.into_iter().map(u16::from).sum()
    }).await.expect("header task completed")
}
