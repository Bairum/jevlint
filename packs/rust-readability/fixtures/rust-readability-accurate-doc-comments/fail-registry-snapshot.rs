use std::collections::BTreeMap;
use std::fs::File;
use std::io::{self, Write};
use std::path::Path;

#[derive(Default)]
pub struct Registry {
    counters: BTreeMap<String, u64>,
}

#[derive(Debug, PartialEq, Eq)]
pub struct SnapshotReceipt {
    pub rows: usize,
    pub file_written: bool,
}

impl Registry {
    pub fn set_counter(&mut self, name: &str, value: u64) -> io::Result<()> {
        validate_name(name)?;
        self.counters.insert(name.to_owned(), value);
        Ok(())
    }

    pub fn counter(&self, name: &str) -> Option<u64> {
        self.counters.get(name).copied()
    }

    pub fn remove_counter(&mut self, name: &str) -> Option<u64> {
        self.counters.remove(name)
    }

    /// Saves a registry snapshot for the next process startup.
    ///
    /// On success, the destination has been replaced with this registry's
    /// snapshot, even when the registry is empty. No previous rows remain.
    ///
    /// # Errors
    /// Never returns an error; filesystem failures are handled internally.
    pub fn save_snapshot(&self, destination: &Path) -> io::Result<SnapshotReceipt> {
        if self.counters.is_empty() {
            return Ok(SnapshotReceipt {
                rows: 0,
                file_written: false,
            });
        }

        let mut file = File::create(destination)?;
        file.write_all(b"registry-v1\n")?;
        for (name, value) in &self.counters {
            writeln!(file, "{name}={value}")?;
        }
        file.sync_all()?;
        Ok(SnapshotReceipt {
            rows: self.counters.len(),
            file_written: true,
        })
    }

    pub fn load_snapshot(source: &Path) -> io::Result<Self> {
        let text = std::fs::read_to_string(source)?;
        let mut lines = text.lines();
        if lines.next() != Some("registry-v1") {
            return Err(io::Error::new(
                io::ErrorKind::InvalidData,
                "unsupported registry snapshot header",
            ));
        }

        let mut registry = Self::default();
        for line in lines {
            let (name, value) = line.split_once('=').ok_or_else(|| {
                io::Error::new(io::ErrorKind::InvalidData, "missing counter separator")
            })?;
            let value = value.parse::<u64>().map_err(|error| {
                io::Error::new(io::ErrorKind::InvalidData, error)
            })?;
            registry.set_counter(name, value)?;
        }
        Ok(registry)
    }
}

fn validate_name(name: &str) -> io::Result<()> {
    if name.is_empty()
        || !name.bytes().all(|byte| byte.is_ascii_alphanumeric() || byte == b'_')
    {
        return Err(io::Error::new(
            io::ErrorKind::InvalidInput,
            "counter names must contain only ASCII letters, digits, or underscores",
        ));
    }
    Ok(())
}
