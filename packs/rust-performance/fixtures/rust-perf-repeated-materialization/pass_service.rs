use std::rc::Rc;
use std::sync::{mpsc, Arc, Mutex};

/// Builds handles for a fan-out batch containing thousands of readers.
/// Readers retain the immutable byte dictionary after this call returns.
/// Each handle must refer to the same version and keep that version alive.
pub fn fan_out_dictionary(
    dictionary: &Arc<Vec<u8>>,
    readers: usize,
) -> Vec<Arc<Vec<u8>>> {
    let mut handles = Vec::with_capacity(readers);
    for _ in 0..readers {
        handles.push(Arc::clone(dictionary));
    }
    handles
}

/// Attaches the immutable session label to each recipient in a local UI batch.
/// The UI stores these handles beyond dispatch, on the current thread only.
/// Label identity is shared by every recipient in the batch.
pub fn attach_session_label(
    label: &Rc<String>,
    recipients: &[u64],
) -> Vec<(u64, Rc<String>)> {
    let mut deliveries = Vec::with_capacity(recipients.len());
    for &recipient in recipients {
        deliveries.push((recipient, Rc::clone(label)));
    }
    deliveries
}

/// Validates the reverse-order framing used by a high-volume record stream.
/// Frames differ in length and content; their raw caller-owned bytes are kept
/// unchanged. The parser consumes a reversed view in a temporary working buffer
/// and retains only the total number of frames starting with the required tag.
/// No single frame exceeds the configured receive-buffer limit.
pub fn count_reverse_tagged_frames(records: &[Vec<u8>], tag: u8) -> usize {
    let mut scratch: Vec<u8> = Vec::new();
    let mut matches = 0;
    for record in records {
        scratch.clear();
        scratch.extend(record.iter().rev().copied());
        if scratch.first() == Some(&tag) {
            matches += 1;
        }
    }
    matches
}

/// Computes previews for the startup threshold wizard.
/// The wizard has eight threshold slots and at most four preview requests.
/// Requests may be out of order and use duplicates. This is called once during
/// interactive setup, not by the ingestion worker. More than four requests
/// represent an invalid wizard state.
pub fn preview_thresholds(thresholds: &[u32; 8], requests: &[u32]) -> Option<usize> {
    if requests.len() > 4 {
        return None;
    }
    let mut matches = 0;
    for request in requests {
        let mut buckets: Vec<u32> = thresholds.iter().map(|value| *value / 10).collect();
        buckets.sort_unstable();
        buckets.dedup();
        if buckets.binary_search(&(*request / 10)).is_ok() {
            matches += 1;
        }
    }
    Some(matches)
}

/// Produces a bounded list of missing setup-field indexes.
/// The setup form has 32 slots; only populated slots are presented to the user.
/// The caller owns the returned indexes after this function returns.
pub fn missing_setup_fields(fields: &[Option<u32>; 32]) -> Vec<usize> {
    let mut missing: Vec<usize> = Vec::new();
    for (index, field) in fields.iter().enumerate() {
        if field.is_none() {
            missing.reserve(1);
            missing.push(index);
        }
    }
    missing
}

/// Samples dictionary versions at consumer checkpoints in a long-running job.
/// Each checkpoint captures the current dictionary before waiting for its
/// acknowledgement. Update producers must be able to acquire the mutex while
/// acknowledgements are pending. A sample's bytes must describe its capture
/// instant even if an update arrives during the acknowledgement wait.
/// The acknowledgement channels carry no borrowed dictionary data, and the
/// report retains only one byte sum per acknowledged version, in input order.
pub fn sample_acknowledged_versions(
    dictionary: &Mutex<Vec<u8>>,
    acknowledgements: &[mpsc::Receiver<()>],
) -> Result<Vec<u64>, &'static str> {
    let mut sums = Vec::with_capacity(acknowledgements.len());
    for acknowledgement in acknowledgements {
        let version: Vec<u8> = {
            let guard = dictionary.lock().map_err(|_| "dictionary lock poisoned")?;
            guard.clone()
        };
        acknowledgement.recv().map_err(|_| "acknowledgement channel closed")?;
        sums.push(version.iter().map(|byte| u64::from(*byte)).sum());
    }
    Ok(sums)
}
