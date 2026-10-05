/// Read administrator configuration. Callers distinguish NotFound from
/// PermissionDenied through the original OS error; the caller already owns path.
pub fn read_config(path: &std::path::Path) -> std::io::Result<String> {
    std::fs::read_to_string(path)
}
