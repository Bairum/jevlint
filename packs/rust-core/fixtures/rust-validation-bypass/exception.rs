/// Raw fields from the billing service; values may be outside local limits.
pub struct PercentRecord {
    pub value: u8,
}

impl PercentRecord {
    pub fn new(value: u8) -> Self {
        Self { value }
    }

    pub fn validated_value(&self) -> Result<u8, &'static str> {
        if self.value > 100 {
            return Err("rate exceeds 100");
        }
        Ok(self.value)
    }

    pub fn encode(&self) -> [u8; 1] {
        [self.value]
    }
}

pub fn forward_record(value: u8) -> [u8; 1] {
    PercentRecord::new(value).encode()
}
