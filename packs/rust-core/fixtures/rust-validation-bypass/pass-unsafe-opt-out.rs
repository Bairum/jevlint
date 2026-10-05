/// A rate in 0..=100, established by validation or the unsafe constructor's obligations.
pub struct Percent {
    value: u8,
}

impl Percent {
    pub fn new(value: u8) -> Result<Self, &'static str> {
        if value > 100 {
            return Err("rate exceeds 100");
        }
        Ok(Self { value })
    }

    /// # Safety
    /// The caller must ensure that value is in 0..=100.
    pub unsafe fn new_unchecked(value: u8) -> Self {
        Self { value }
    }

    /// The remaining rate is in 0..=100.
    pub fn complement(&self) -> i16 {
        100 - i16::from(self.value)
    }
}

pub fn configured_remaining_rate() -> i16 {
    // SAFETY: the configured literal is in the required range.
    let rate = unsafe { Percent::new_unchecked(25) };
    rate.complement()
}
