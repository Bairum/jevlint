/// Every safe RegionCode is one of the three configured region indices.
pub struct RegionCode {
    index: u8,
}

impl RegionCode {
    pub fn new(index: u8) -> Result<Self, &'static str> {
        if index >= 3 {
            return Err("unknown region");
        }
        Ok(Self { index })
    }

    pub fn from_wire(index: u8) -> Self {
        Self::from(index)
    }

    /// Returns the configured region name for every RegionCode.
    pub fn name(&self) -> &'static str {
        ["west", "central", "east"][usize::from(self.index)]
    }
}

impl From<u8> for RegionCode {
    fn from(index: u8) -> Self {
        Self { index }
    }
}

pub fn wire_region_name(index: u8) -> &'static str {
    RegionCode::from_wire(index).name()
}
