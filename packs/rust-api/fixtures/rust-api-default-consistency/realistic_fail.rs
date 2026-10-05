/// New and Default select the scheduler's enabled, three-attempt baseline.
pub struct SchedulerOptions {
    pub enabled: bool,
    pub attempts: u8,
    pub batch_limit: usize,
}

impl SchedulerOptions {
    pub fn new() -> Self {
        Self {
            enabled: true,
            attempts: 3,
            batch_limit: 16,
        }
    }
}

impl Default for SchedulerOptions {
    fn default() -> Self {
        Self {
            enabled: false,
            attempts: 0,
            batch_limit: 16,
        }
    }
}

pub struct WorkItem {
    pub id: u64,
    pub payload: String,
    pub attempt: u8,
}

pub struct Scheduler {
    options: SchedulerOptions,
    pending: std::collections::VecDeque<WorkItem>,
    completed: Vec<u64>,
}

impl Scheduler {
    pub fn with_options(options: SchedulerOptions) -> Self {
        Self {
            options,
            pending: std::collections::VecDeque::new(),
            completed: Vec::new(),
        }
    }

    pub fn enqueue(&mut self, id: u64, payload: String) {
        self.pending.push_back(WorkItem {
            id,
            payload,
            attempt: 0,
        });
    }

    pub fn next_batch(&mut self) -> Vec<WorkItem> {
        if !self.options.enabled {
            return Vec::new();
        }
        let count = self.options.batch_limit.min(self.pending.len());
        self.pending.drain(..count).collect()
    }

    pub fn complete(&mut self, item: WorkItem) {
        self.completed.push(item.id);
    }

    pub fn retry(&mut self, mut item: WorkItem) -> bool {
        item.attempt = item.attempt.saturating_add(1);
        if item.attempt >= self.options.attempts {
            return false;
        }
        self.pending.push_back(item);
        true
    }

    pub fn pending_count(&self) -> usize {
        self.pending.len()
    }

    pub fn completed_ids(&self) -> &[u64] {
        &self.completed
    }
}

pub fn restore_pending(
    options: SchedulerOptions,
    records: impl IntoIterator<Item = (u64, String)>,
) -> Scheduler {
    let mut scheduler = Scheduler::with_options(options);
    for (id, payload) in records {
        scheduler.enqueue(id, payload);
    }
    scheduler
}
