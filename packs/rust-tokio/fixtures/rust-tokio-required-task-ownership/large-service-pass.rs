use std::io;
use std::sync::Arc;
use tokio::sync::{mpsc, RwLock};

#[derive(Clone)]
pub struct Registration {
    pub account: String,
    pub sequence: u64,
    pub enabled: bool,
}

pub struct Receipt {
    pub account: String,
    pub sequence: u64,
}

pub struct Registry {
    pub outgoing: mpsc::Sender<Receipt>,
    pub enabled_accounts: Arc<RwLock<Vec<String>>>,
}

pub fn parse_registration(line: &str) -> io::Result<Registration> {
    let mut fields = line.split(',');
    let account = fields.next().unwrap_or_default().trim();
    if account.is_empty() {
        return Err(io::Error::new(io::ErrorKind::InvalidInput, "empty account"));
    }
    let sequence = fields.next().unwrap_or_default().parse::<u64>()
        .map_err(|error| io::Error::new(io::ErrorKind::InvalidInput, error))?;
    let enabled = match fields.next().unwrap_or_default().trim() {
        "enabled" => true,
        "disabled" => false,
        _ => return Err(io::Error::new(io::ErrorKind::InvalidInput, "unknown state")),
    };
    if fields.next().is_some() {
        return Err(io::Error::new(io::ErrorKind::InvalidInput, "extra fields"));
    }
    Ok(Registration { account: account.to_owned(), sequence, enabled })
}

pub fn render_receipt(receipt: &Receipt) -> String {
    format!("{}:{}", receipt.account, receipt.sequence)
}

pub async fn allowed(registry: &Registry, account: &str) -> bool {
    let accounts = registry.enabled_accounts.read().await;
    accounts.iter().any(|candidate| candidate == account)
}

pub async fn replace_accounts(registry: &Registry, accounts: Vec<String>) {
    let mut current = registry.enabled_accounts.write().await;
    *current = accounts;
}

async fn publish(outgoing: mpsc::Sender<Receipt>, registration: Registration) -> io::Result<()> {
    outgoing.send(Receipt {
        account: registration.account,
        sequence: registration.sequence,
    }).await.map_err(|_| io::Error::new(io::ErrorKind::BrokenPipe, "receipt sink closed"))
}

/// Success acknowledges delivery of every enabled registration's receipt.
/// Delivery errors and task failures are returned to the request owner.
pub async fn process_batch(registry: &Registry, lines: &[String]) -> io::Result<usize> {
    let mut accepted = 0;
    for line in lines {
        let registration = parse_registration(line)?;
        if !registration.enabled || !allowed(registry, &registration.account).await {
            continue;
        }
        let outgoing = registry.outgoing.clone();
        let task = tokio::spawn(publish(outgoing, registration));
        task.await.map_err(io::Error::other)??;
        accepted += 1;
    }
    Ok(accepted)
}

/// Audit output is disposable and does not affect registration delivery.
pub async fn audit(registry: &Registry, destination: mpsc::Sender<String>) {
    let accounts = registry.enabled_accounts.read().await.clone();
    let task = tokio::spawn(async move {
        for account in accounts {
            if destination.try_send(account).is_err() {
                break;
            }
        }
    });
    drop(task);
}

pub async fn snapshot(registry: &Registry) -> Vec<String> {
    registry.enabled_accounts.read().await.clone()
}
