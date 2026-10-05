use std::path::PathBuf;
use std::time::Duration;

pub struct Request {
    pub id: u64,
    pub path: PathBuf,
    pub maximum_bytes: usize,
}

pub struct Reply {
    pub id: u64,
    pub body: Vec<u8>,
    pub checksum: u64,
}

pub struct ServicePolicy {
    pub retry_delay: Duration,
    pub maximum_bytes: usize,
}

impl ServicePolicy {
    pub fn standard() -> Self {
        Self {
            retry_delay: Duration::from_millis(25),
            maximum_bytes: 1024 * 1024,
        }
    }

    pub fn accepts(&self, request: &Request) -> bool {
        request.maximum_bytes <= self.maximum_bytes
    }
}

pub fn checksum(bytes: &[u8]) -> u64 {
    bytes.iter().fold(0_u64, |sum, byte| {
        sum.wrapping_add(u64::from(*byte))
    })
}

pub fn make_reply(id: u64, bytes: Vec<u8>) -> Reply {
    Reply {
        id,
        checksum: checksum(&bytes),
        body: bytes,
    }
}

pub fn enforce_limit(bytes: Vec<u8>, maximum: usize) -> std::io::Result<Vec<u8>> {
    if bytes.len() > maximum {
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidData,
            "response exceeds request limit",
        ));
    }
    Ok(bytes)
}

pub async fn retry_pause(policy: &ServicePolicy) {
    tokio::time::sleep(policy.retry_delay).await;
}

pub async fn notify_ready(sender: tokio::sync::oneshot::Sender<()>) {
    let _ = sender.send(());
}

/// Requests share the current-thread runtime with service timers.
pub fn serve_request(request: Request) -> std::io::Result<Reply> {
    let runtime = tokio::runtime::Builder::new_current_thread()
        .enable_time()
        .build()?;
    runtime.block_on(async move {
        tokio::task::spawn(async move {
            let policy = ServicePolicy::standard();
            if !policy.accepts(&request) {
                return Err(std::io::Error::new(
                    std::io::ErrorKind::InvalidInput,
                    "requested response limit is unsupported",
                ));
            }
            let id = request.id;
            let maximum = request.maximum_bytes;
            let bytes = std::fs::read(request.path)?;
            let bytes = enforce_limit(bytes, maximum)?;
            Ok(make_reply(id, bytes))
        })
        .await
        .map_err(|error| std::io::Error::new(std::io::ErrorKind::Other, error))?
    })
}
