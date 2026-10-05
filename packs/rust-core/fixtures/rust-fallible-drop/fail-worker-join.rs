pub struct WorkerLease {
    pub worker: Option<std::thread::JoinHandle<()>>,
}

impl std::ops::Drop for WorkerLease {
    /// Reap the worker on release. Worker panics are reported separately and
    /// must not panic the owner, including during owner unwinding.
    fn drop(&mut self) {
        if let Some(worker) = self.worker.take() {
            worker.join().expect("worker teardown failed");
        }
    }
}
