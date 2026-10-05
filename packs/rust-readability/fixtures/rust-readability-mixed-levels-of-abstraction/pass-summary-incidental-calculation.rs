use std::fs;
use std::io;
use std::path::{Path, PathBuf};

pub struct Summary {
    pub window_start_secs: u64,
    pub window_end_secs: u64,
    pub total_samples: u64,
}

pub fn publish_summary(directory: &Path, summary: &Summary) -> io::Result<PathBuf> {
    if summary.window_end_secs < summary.window_start_secs {
        return Err(io::Error::new(io::ErrorKind::InvalidInput, "reversed window"));
    }
    fs::create_dir_all(directory)?;
    let duration_secs = summary.window_end_secs - summary.window_start_secs;
    let path = directory.join(format!("summary-{}.txt", summary.window_end_secs));
    let text = format!(
        "duration_secs={duration_secs}\nsamples={}\n",
        summary.total_samples,
    );
    fs::write(&path, text)?;
    Ok(path)
}
