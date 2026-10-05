/// One executor consumes an externally driven bounded queue.
/// Message size is capped at 256 bytes by the producer; there is no per-message task.
pub async fn run_executor(mut queue: tokio::sync::mpsc::Receiver<[u8; 256]>) -> u64 {
    let mut total = 0_u64;
    while let Some(bytes) = queue.recv().await {
        total = bytes.iter().fold(total, |sum, byte| sum.wrapping_add(u64::from(*byte)));
        tokio::task::yield_now().await;
    }
    total
}
