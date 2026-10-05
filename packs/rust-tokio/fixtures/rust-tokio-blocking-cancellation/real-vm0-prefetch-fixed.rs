use std::io::{Cursor, Read};
use std::sync::atomic::{AtomicBool, AtomicUsize, Ordering};
use std::sync::{Arc, Mutex};

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum RunnerMode {
    Starting,
    Running,
    Stopped,
}

pub struct Snapshot {
    pub name: String,
    pub bytes: Vec<u8>,
}

pub struct PrefetchReport {
    pub snapshot_name: String,
    pub warmed_bytes: usize,
    pub mode: RunnerMode,
}

pub struct StartupPolicy {
    pub chunk_bytes: usize,
    pub enabled: bool,
}

impl StartupPolicy {
    pub fn standard() -> Self {
        Self {
            chunk_bytes: 4096,
            enabled: true,
        }
    }

    pub fn accepts(&self, snapshot: &Snapshot) -> bool {
        self.enabled && !snapshot.bytes.is_empty() && self.chunk_bytes > 0
    }
}

pub fn prepare_snapshot(name: &str, bytes: Vec<u8>) -> Snapshot {
    Snapshot {
        name: name.to_owned(),
        bytes,
    }
}

pub fn report(snapshot_name: String, warmed_bytes: usize, mode: RunnerMode) -> PrefetchReport {
    PrefetchReport {
        snapshot_name,
        warmed_bytes,
        mode,
    }
}

pub async fn announce_running(mode: &Arc<Mutex<RunnerMode>>) {
    tokio::task::yield_now().await;
    *mode.lock().unwrap() = RunnerMode::Running;
}

/// RunnerMode::Stopped means all startup prefetch I/O has ended.
/// The runner owns one prefetch worker and stops it before publishing Stopped.
pub async fn shutdown_prefetch(snapshot: Snapshot) -> std::io::Result<PrefetchReport> {
    let policy = StartupPolicy::standard();
    let name = snapshot.name.clone();
    let mode = Arc::new(Mutex::new(RunnerMode::Starting));
    if !policy.accepts(&snapshot) {
        return Ok(report(name, 0, RunnerMode::Stopped));
    }
    announce_running(&mode).await;
    let warmed = Arc::new(AtomicUsize::new(0));
    let worker_warmed = warmed.clone();
    let stop = Arc::new(AtomicBool::new(false));
    let worker_stop = stop.clone();
    let (started_sender, started_receiver) = tokio::sync::oneshot::channel();
    let (release_sender, release_receiver) = std::sync::mpsc::channel();
    let worker = tokio::task::spawn_blocking(move || -> std::io::Result<()> {
        let mut input = Cursor::new(snapshot.bytes);
        let mut chunk = vec![0_u8; policy.chunk_bytes];
        started_sender.send(()).unwrap();
        release_receiver.recv().unwrap();
        loop {
            if worker_stop.load(Ordering::SeqCst) {
                break;
            }
            let count = input.read(&mut chunk)?;
            if count == 0 {
                break;
            }
            worker_warmed.fetch_add(count, Ordering::SeqCst);
        }
        Ok(())
    });
    started_receiver.await.unwrap();
    stop.store(true, Ordering::SeqCst);
    release_sender.send(()).unwrap();
    worker.await.unwrap()?;
    *mode.lock().unwrap() = RunnerMode::Stopped;
    let completed_mode = *mode.lock().unwrap();
    Ok(report(name, warmed.load(Ordering::SeqCst), completed_mode))
}
