#[derive(Debug, PartialEq, Eq)]
pub enum RegistrationRejection {
    EmptyName,
    ReservedName,
}

/// Reject names outside the registration policy. There is no underlying
/// operation error; rejection describes the complete domain decision.
pub fn register_name(name: &str) -> Result<String, RegistrationRejection> {
    if name.trim().is_empty() {
        return Err(RegistrationRejection::EmptyName);
    }
    if name.eq_ignore_ascii_case("administrator") {
        return Err(RegistrationRejection::ReservedName);
    }
    Ok(name.to_owned())
}
