use std::num::NonZeroUsize;
use std::sync::Arc;
use tokio::sync::{mpsc, Semaphore};
use tokio::task::JoinSet;

pub struct Job {
    pub id: u64,
    pub samples: [u16; 128],
}

pub struct Summary {
    pub id: u64,
    pub minimum: u16,
    pub maximum: u16,
    pub average: u32,
}

pub struct WorkerConfig {
    pub task_limit: NonZeroUsize,
    pub label: String,
}

pub fn parse_config(label: &str, limit: &str) -> Result<WorkerConfig, String> {
    if label.trim().is_empty() {
        return Err("worker label required".to_owned());
    }
    let parsed = limit.parse::<usize>().map_err(|error| error.to_string())?;
    let task_limit = NonZeroUsize::new(parsed).ok_or("positive limit required")?;
    Ok(WorkerConfig { task_limit, label: label.trim().to_owned() })
}

pub fn summarize(job: Job) -> Summary {
    let mut minimum = u16::MAX;
    let mut maximum = 0;
    let mut sum = 0_u32;
    for value in job.samples {
        minimum = minimum.min(value);
        maximum = maximum.max(value);
        sum += u32::from(value);
    }
    Summary { id: job.id, minimum, maximum, average: sum / 128 }
}

pub fn encode(summary: &Summary) -> String {
    format!("{},{},{},{}", summary.id, summary.minimum, summary.maximum, summary.average)
}

pub fn sample_job(id: u64, value: u16) -> Job {
    Job { id, samples: [value; 128] }
}

async fn report(summary: Summary, destination: mpsc::Sender<String>) {
    let _ = destination.send(encode(&summary)).await;
}

/// External producers may submit measurements continuously.
/// `task_limit` caps live measurement tasks, including permit waiters and reporting tasks.
/// Measurement reports are disposable; the input queue has a separate upstream budget.
pub async fn run_pipeline(mut input: mpsc::UnboundedReceiver<Job>, destination: mpsc::Sender<String>, config: WorkerConfig) {
    let capacity = Arc::new(Semaphore::new(config.task_limit.get()));
    let mut tasks = JoinSet::new();
    while let Some(job) = input.recv().await {
        let permit = capacity.clone().acquire_owned().await.expect("capacity open");
        let destination = destination.clone();
        tasks.spawn(async move {
            tokio::task::yield_now().await;
            report(summarize(job), destination).await;
            drop(permit);
        });
        while let Some(result) = tasks.try_join_next() {
            result.expect("measurement task joined");
        }
    }
    while let Some(result) = tasks.join_next().await {
        result.expect("measurement task joined");
    }
}

/// A calibration batch is finite; capacity only limits simultaneous computations.
pub async fn calibrate(jobs: Vec<Job>, capacity: Arc<Semaphore>) -> Vec<Summary> {
    let mut tasks = JoinSet::new();
    for job in jobs {
        let capacity = capacity.clone();
        tasks.spawn(async move {
            let _permit = capacity.acquire_owned().await.expect("capacity open");
            summarize(job)
        });
    }
    let mut summaries = Vec::new();
    while let Some(result) = tasks.join_next().await {
        summaries.push(result.expect("calibration task joined"));
    }
    summaries.sort_by_key(|summary| summary.id);
    summaries
}

pub async fn serial_report(jobs: Vec<Job>, destination: mpsc::Sender<String>) {
    for job in jobs {
        report(summarize(job), destination.clone()).await;
    }
}
