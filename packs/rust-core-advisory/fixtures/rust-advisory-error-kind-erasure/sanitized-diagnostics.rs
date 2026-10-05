use std::io;
use std::path::{Path, PathBuf};

#[derive(Debug)]
pub struct InternalDiagnostic {
    pub request_id: u64,
    pub resource: PathBuf,
    pub cause: io::Error,
}

#[derive(Debug)]
pub struct PublicFailure {
    pub request_id: u64,
}

/// Sends resource details only to the trusted diagnostic sink. The response
/// contains an incident reference, never tenant paths or OS error messages.
pub fn tenant_document(
    path: &Path,
    request_id: u64,
    diagnostics: &mut Vec<InternalDiagnostic>,
) -> Result<String, PublicFailure> {
    std::fs::read_to_string(path).map_err(|cause| {
        diagnostics.push(InternalDiagnostic {
            request_id,
            resource: path.to_owned(),
            cause,
        });
        PublicFailure { request_id }
    })
}
