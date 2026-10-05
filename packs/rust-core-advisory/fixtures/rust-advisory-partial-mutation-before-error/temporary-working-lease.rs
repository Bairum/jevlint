use std::num::ParseIntError;

/// Retains scratch capacity between calls, never published readings.
/// The private scratch buffer is empty whenever no working lease exists.
#[derive(Default)]
pub struct ReadingWorkspace {
    scratch: Vec<u32>,
}

/// Temporary work only: dropping the lease clears all entries, including on Err.
struct WorkingLease<'a> {
    scratch: &'a mut Vec<u32>,
}

impl Drop for WorkingLease<'_> {
    fn drop(&mut self) {
        self.scratch.clear();
    }
}

/// Parsed prefixes belong only to this temporary lease and are cleared by Drop
/// if parsing fails; they are not retained application readings.
fn fill_lease(lease: &mut WorkingLease<'_>, fields: &[&str]) -> Result<(), ParseIntError> {
    for field in fields {
        lease.scratch.push(field.parse()?);
    }
    Ok(())
}

impl ReadingWorkspace {
    pub fn median_reading(&mut self, fields: &[&str]) -> Result<Option<u32>, ParseIntError> {
        let mut lease = WorkingLease {
            scratch: &mut self.scratch,
        };
        fill_lease(&mut lease, fields)?;
        lease.scratch.sort_unstable();
        Ok(lease.scratch.get(lease.scratch.len() / 2).copied())
    }
}
