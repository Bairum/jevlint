use std::sync::Arc;
use tokio::sync::Mutex;

/// Adds the worker's contribution to the account and returns its balance.
pub async fn settle_account(account: Arc<Mutex<u64>>) -> u64 {
    let mut balance = account.lock().await;
    let child_account = Arc::clone(&account);
    let child = tokio::spawn(async move { credit_account(child_account).await });
    let contribution = child.await.expect("credit task panicked");
    *balance += contribution;
    *balance
}

async fn credit_account(account: Arc<Mutex<u64>>) -> u64 {
    let mut balance = account.lock().await;
    *balance += 10;
    2
}
