use std::fs::OpenOptions;
use std::io::{self, Write};
use std::path::Path;
use std::sync::Mutex;

static RECORDS: Mutex<()> = Mutex::new(());

pub fn claim_shared_record(directory: &Path, id: u64, payload: &[u8]) -> io::Result<()> {
    let _guard = RECORDS.lock().map_err(|_| io::Error::other("record lock poisoned"))?;
    let path = directory.join(format!("{id}.record"));
    if path.try_exists()? {
        return Err(io::Error::new(io::ErrorKind::AlreadyExists, "record exists"));
    }
    let mut file = OpenOptions::new()
        .write(true)
        .create(true)
        .truncate(true)
        .open(&path)?;
    file.write_all(payload)?;
    file.sync_all()
}
