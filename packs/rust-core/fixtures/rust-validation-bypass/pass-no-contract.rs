#[derive(Clone, Copy)]
pub struct Percent {
    pub value: u8,
}

impl Percent {
    pub fn new(value: u8) -> Result<Self, &'static str> {
        if value > 100 {
            return Err("rate exceeds 100");
        }
        Ok(Self { value })
    }

    pub fn complement(&self) -> i16 {
        100 - i16::from(self.value)
    }

    pub fn value(&self) -> u8 {
        self.value
    }
}

pub fn remaining_rate(raw: u8) -> i16 {
    let rate = Percent { value: raw };
    rate.complement()
}
