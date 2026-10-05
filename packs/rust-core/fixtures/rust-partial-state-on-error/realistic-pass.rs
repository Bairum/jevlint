#[derive(Clone, Debug, PartialEq, Eq)]
pub struct DeviceProfile {
    pub label: String,
    pub port: u16,
    pub mode: Mode,
    pub revision: u64,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Mode {
    Active,
    Standby,
}

pub struct ProfileUpdate<'a> {
    pub label: &'a str,
    pub port: &'a str,
    pub mode: &'a str,
}

pub fn default_profile() -> DeviceProfile {
    DeviceProfile {
        label: String::from("gateway"),
        port: 8080,
        mode: Mode::Standby,
        revision: 0,
    }
}

pub fn parse_mode(value: &str) -> std::io::Result<Mode> {
    match value {
        "active" => Ok(Mode::Active),
        "standby" => Ok(Mode::Standby),
        _ => Err(std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "unknown operating mode",
        )),
    }
}

pub fn parse_port(value: &str) -> std::io::Result<u16> {
    let port = str::parse::<u16>(value).map_err(|error| {
        std::io::Error::new(std::io::ErrorKind::InvalidInput, error)
    })?;
    if port == 0 {
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "port must be nonzero",
        ));
    }
    Ok(port)
}

pub fn validate_label(value: &str) -> std::io::Result<()> {
    if value.is_empty() || value.len() > 64 || value.contains('\n') {
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "label must occupy one nonempty line of at most 64 bytes",
        ));
    }
    Ok(())
}

pub fn mode_name(mode: Mode) -> &'static str {
    match mode {
        Mode::Active => "active",
        Mode::Standby => "standby",
    }
}

pub fn profile_snapshot(profile: &DeviceProfile) -> DeviceProfile {
    profile.clone()
}

pub fn display_profile(profile: &DeviceProfile) -> String {
    format!(
        "{}:{}:{}:{}",
        profile.label,
        profile.port,
        mode_name(profile.mode),
        profile.revision,
    )
}

pub fn profile_is_enabled(profile: &DeviceProfile) -> bool {
    profile.mode == Mode::Active && profile.port != 0
}

/// Apply one administrative update atomically. Any Err leaves the entire
/// profile unchanged; readers and subsequent updates may reuse it after Err.
pub fn apply_profile(profile: &mut DeviceProfile, update: ProfileUpdate<'_>) -> std::io::Result<()> {
    validate_label(update.label)?;
    let port = parse_port(update.port)?;
    let mode = parse_mode(update.mode)?;
    let revision = profile.revision.checked_add(1).ok_or_else(|| {
        std::io::Error::new(std::io::ErrorKind::InvalidInput, "revision exhausted")
    })?;
    profile.label.clear();
    profile.label.push_str(update.label);
    profile.port = port;
    profile.mode = mode;
    profile.revision = revision;
    Ok(())
}
