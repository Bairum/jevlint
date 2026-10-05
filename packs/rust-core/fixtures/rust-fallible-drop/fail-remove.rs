pub struct UploadLease {
    pub path: std::path::PathBuf,
}

impl std::ops::Drop for UploadLease {
    /// Delete the temporary upload on release. Concurrent retention jobs may
    /// already have removed it; filesystem cleanup errors must not panic.
    fn drop(&mut self) {
        if let Err(error) = std::fs::remove_file(&self.path) {
            panic!("upload cleanup failed: {error}");
        }
    }
}
