use std::fs::OpenOptions;
use std::io::{BufWriter, Write};
use std::path::Path;

/// Builds image bootstrap metadata for bulk publication to the container registry.
/// Production images contain 100,000 preencoded 128-byte inode metadata entries.
/// Build throughput is the priority; readers receive the path after completion.
/// No reader requires an individual inode entry to be delivered during the build.
pub fn dump_bootstrap(
    bootstrap_path: &Path,
    entries: &[[u8; 128]; 100_000],
) -> std::io::Result<usize> {
    let mut bootstrap = BufWriter::with_capacity(
        2 << 17,
        OpenOptions::new()
            .write(true)
            .create(true)
            .truncate(true)
            .open(bootstrap_path)?,
    );
    for entry in entries {
        bootstrap.write_all(entry)?;
    }
    bootstrap.flush()?;
    Ok(entries.len())
}
