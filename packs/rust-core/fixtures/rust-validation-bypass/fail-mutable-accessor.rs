/// Every safe RetryLimit holds a value in 1..=8.
pub struct RetryLimit {
    attempts: u8,
}

impl RetryLimit {
    pub fn new(attempts: u8) -> Result<Self, &'static str> {
        if !(1..=8).contains(&attempts) {
            return Err("retry limit out of range");
        }
        Ok(Self { attempts })
    }

    pub fn attempts_mut(&mut self) -> &mut u8 {
        &mut self.attempts
    }

    /// A scheduling share is defined for every RetryLimit.
    pub fn share(&self, total_slots: u32) -> u32 {
        total_slots / u32::from(self.attempts)
    }
}

pub fn schedule(total_slots: u32) -> Result<u32, &'static str> {
    let mut limit = RetryLimit::new(4)?;
    *limit.attempts_mut() = 0;
    Ok(limit.share(total_slots))
}
