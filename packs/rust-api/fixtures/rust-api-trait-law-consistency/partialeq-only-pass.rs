pub struct Measurement {
    pub value: f64,
}

impl Measurement {
    pub fn scaled(&self, factor: f64) -> Self {
        Self {
            value: self.value * factor,
        }
    }
}

impl std::cmp::PartialEq for Measurement {
    fn eq(&self, other: &Self) -> bool {
        self.value == other.value
    }
}
