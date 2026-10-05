/// Worker count is nonzero for every safely constructed or edited pool.
pub struct WorkerPool {
    pub workers: std::num::NonZeroUsize,
}

impl WorkerPool {
    pub fn new(workers: usize) -> Result<Self, &'static str> {
        let workers = std::num::NonZeroUsize::new(workers).ok_or("workers must be nonzero")?;
        Ok(Self { workers })
    }

    pub fn workers_mut(&mut self) -> &mut std::num::NonZeroUsize {
        &mut self.workers
    }

    pub fn share(&self, jobs: usize) -> usize {
        jobs / self.workers.get()
    }
}

pub fn distribute(jobs: usize) -> Result<usize, &'static str> {
    let mut pool = WorkerPool::new(4)?;
    *pool.workers_mut() = std::num::NonZeroUsize::new(2).ok_or("invalid worker count")?;
    Ok(pool.share(jobs))
}
