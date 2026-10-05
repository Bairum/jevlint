pub struct SensorPosition {
    pub millimeters: i32,
}

impl SensorPosition {
    pub fn meters(&self) -> f64 {
        f64::from(self.millimeters) / 1000.0
    }
}

impl std::cmp::PartialEq for SensorPosition {
    fn eq(&self, other: &Self) -> bool {
        self.millimeters.abs_diff(other.millimeters) <= 1
    }
}

impl std::cmp::Eq for SensorPosition {}
