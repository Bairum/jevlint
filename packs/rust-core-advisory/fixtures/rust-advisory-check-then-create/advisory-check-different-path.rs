use std::fs;
use std::io;
use std::path::Path;

/// Always replaces the status file; the input observation is informational.
pub fn publish_input_status(input: &Path, status: &Path) -> io::Result<()> {
    let state = if input.exists() { "present\n" } else { "absent\n" };
    fs::write(status, state)
}
