use std::sync::Arc;
use tokio::sync::Mutex;

pub async fn settle_account(account: Arc<Mutex<u64>>) -> u64 {
    {
        let mut balance = account.lock().await;
        *balance += 2;
    }
    let child_account = Arc::clone(&account);
    let child = tokio::spawn(async move { credit_account(child_account).await });
    child.await.expect("credit task panicked");
    let balance = account.lock().await;
    *balance
}

async fn credit_account(account: Arc<Mutex<u64>>) {
    let mut balance = account.lock().await;
    *balance += 10;
}
