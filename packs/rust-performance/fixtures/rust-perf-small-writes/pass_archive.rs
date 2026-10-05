use std::fs::{self, File};
use std::io::{self, BufWriter, Read, Write};
use std::path::{Path, PathBuf};

#[derive(Clone, Copy)]
pub struct Event {
    pub sequence: u64,
    pub source: u16,
    pub value: i32,
}

pub struct ArchivePaths {
    pub pending: PathBuf,
    pub published: PathBuf,
    pub manifest: PathBuf,
}

pub struct ArchiveSummary {
    pub records: usize,
    pub first_sequence: u64,
    pub last_sequence: u64,
}

pub fn validate_events(events: &[Event; 100_000]) -> io::Result<ArchiveSummary> {
    let mut previous = events[0].sequence;
    for event in &events[1..] {
        if event.sequence <= previous {
            return Err(io::Error::new(
                io::ErrorKind::InvalidInput,
                "event sequences must increase",
            ));
        }
        previous = event.sequence;
    }
    Ok(ArchiveSummary {
        records: events.len(),
        first_sequence: events[0].sequence,
        last_sequence: previous,
    })
}

/// Encodes a batch in memory for the compression worker.
/// Compression begins after the complete byte vector is available.
pub fn encode_binary(events: &[Event; 100_000]) -> io::Result<Vec<u8>> {
    let mut bytes = Vec::with_capacity(events.len() * 14);
    for event in events {
        bytes.write_all(&event.sequence.to_le_bytes())?;
        bytes.write_all(&event.source.to_le_bytes())?;
        bytes.write_all(&event.value.to_le_bytes())?;
    }
    Ok(bytes)
}

/// Writes the nightly 100,000-event archive for throughput-oriented bulk ingestion.
/// Typical rows occupy 20 to 40 bytes. The ingestion worker opens only published
/// archives and has no per-row visibility or latency requirement.
/// The caller publishes the pending path only after this operation succeeds.
pub fn export_pending(path: &Path, events: &[Event; 100_000]) -> io::Result<()> {
    let file = File::create(path)?;
    let mut output = BufWriter::new(file);
    for event in events {
        output.write_fmt(format_args!(
            "{},{},{}\n",
            event.sequence,
            event.source,
            event.value,
        ))?;
    }
    output.flush()?;
    Ok(())
}

/// Writes the three fields of a single archive-level manifest.
pub fn write_manifest(path: &Path, summary: &ArchiveSummary) -> io::Result<()> {
    let fields = [
        format!("records={}\n", summary.records),
        format!("first={}\n", summary.first_sequence),
        format!("last={}\n", summary.last_sequence),
    ];
    let mut output = File::create(path)?;
    for field in &fields {
        output.write_all(field.as_bytes())?;
    }
    Ok(())
}

pub fn read_manifest(path: &Path) -> io::Result<String> {
    let mut input = File::open(path)?;
    let mut text = String::new();
    input.read_to_string(&mut text)?;
    Ok(text)
}

/// Returns only after the complete archive has been renamed into the import queue.
/// A failed operation must not expose the pending file to the ingestion worker.
pub fn publish_archive(
    paths: &ArchivePaths,
    events: &[Event; 100_000],
) -> io::Result<ArchiveSummary> {
    let summary = validate_events(events)?;
    if let Err(error) = export_pending(&paths.pending, events) {
        let _ = fs::remove_file(&paths.pending);
        return Err(error);
    }
    write_manifest(&paths.manifest, &summary)?;
    fs::rename(&paths.pending, &paths.published)?;
    Ok(summary)
}
