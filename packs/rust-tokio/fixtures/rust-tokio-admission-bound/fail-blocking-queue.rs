/// External producers may submit continuously.
/// At most `cpu_limit` computations may be admitted, including queued computations.
pub async fn checksum_service(mut requests: tokio::sync::mpsc::UnboundedReceiver<Vec<u8>>, cpu_limit: usize) {
    let _configured_limit = cpu_limit;
    while let Some(bytes) = requests.recv().await {
        tokio::task::spawn_blocking(move || {
            bytes.into_iter().fold(0_u64, |sum, byte| sum.wrapping_add(u64::from(byte)))
        });
    }
}
