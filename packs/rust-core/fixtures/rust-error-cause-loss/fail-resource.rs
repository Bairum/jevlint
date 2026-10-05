#[derive(Debug)]
pub struct ReadFailure {
    pub resource: std::path::PathBuf,
    pub cause: std::io::Error,
}

/// Load the named tenant profile. On failure, resource identifies the exact
/// attempted file, and cause retains its OS error for repair instructions.
pub fn load_profile(root: &std::path::Path, tenant: &str) -> Result<String, ReadFailure> {
    let resource = root.join(tenant).join("profile.conf");
    std::fs::read_to_string(&resource).map_err(|cause| ReadFailure {
        resource: root.to_path_buf(),
        cause,
    })
}
