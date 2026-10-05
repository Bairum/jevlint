pub struct TemperatureReading {
    pub celsius: f64,
}

impl TemperatureReading {
    pub fn fahrenheit(&self) -> f64 {
        self.celsius * 1.8 + 32.0
    }
}

impl std::cmp::PartialEq for TemperatureReading {
    fn eq(&self, other: &Self) -> bool {
        self.celsius == other.celsius
    }
}

impl std::cmp::Eq for TemperatureReading {}
