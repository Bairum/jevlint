/// An editable input record, not an approved region identifier.
pub struct RegionDraft {
    pub index: u8,
}

impl RegionDraft {
    pub fn new(index: u8) -> Self {
        Self { index }
    }

    pub fn validate(self) -> Result<RegionCode, &'static str> {
        RegionCode::new(self.index)
    }
}

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

    /// Returns the configured region name for every RegionCode.
    pub fn name(&self) -> &'static str {
        ["west", "central", "east"][usize::from(self.index)]
    }
}

pub fn lookup_draft(mut draft: RegionDraft, selected: u8) -> Result<&'static str, &'static str> {
    draft.index = selected;
    let region = draft.validate()?;
    Ok(region.name())
}
