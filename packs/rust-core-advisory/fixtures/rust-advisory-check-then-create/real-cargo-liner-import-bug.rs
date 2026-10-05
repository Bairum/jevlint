use std::collections::BTreeMap;
use std::fs;
use std::io;
use std::path::Path;

#[derive(Clone, Copy)]
pub enum VersionPolicy {
    Exact,
    Compatible,
    Any,
}

pub struct InstalledPackage {
    pub name: String,
    pub version: String,
}

pub struct Configuration {
    packages: BTreeMap<String, String>,
}

fn validate_package(package: &InstalledPackage) -> io::Result<()> {
    if package.name.is_empty()
        || !package.name.bytes().all(|byte| {
            byte.is_ascii_alphanumeric() || byte == b'-' || byte == b'_'
        })
    {
        return Err(io::Error::new(io::ErrorKind::InvalidInput, "invalid package name"));
    }
    let mut parts = package.version.split('.');
    for _ in 0..3 {
        match parts.next() {
            Some(part) if !part.is_empty() && part.bytes().all(|byte| byte.is_ascii_digit()) => {}
            _ => return Err(io::Error::new(io::ErrorKind::InvalidInput, "invalid package version")),
        }
    }
    if parts.next().is_some() {
        return Err(io::Error::new(io::ErrorKind::InvalidInput, "invalid package version"));
    }
    Ok(())
}

pub fn build_configuration(
    installed: Vec<InstalledPackage>,
    policy: VersionPolicy,
    keep_importer: bool,
) -> io::Result<Configuration> {
    let mut packages = BTreeMap::new();
    for package in installed {
        validate_package(&package)?;
        if !keep_importer && package.name == "cargo-liner" {
            continue;
        }
        let requirement = match policy {
            VersionPolicy::Exact => format!("={}", package.version),
            VersionPolicy::Compatible => format!("^{}", package.version),
            VersionPolicy::Any => String::from("*"),
        };
        if packages.insert(package.name, requirement).is_some() {
            return Err(io::Error::new(io::ErrorKind::InvalidInput, "duplicate package"));
        }
    }
    Ok(Configuration { packages })
}

impl Configuration {
    pub fn render(&self) -> String {
        let mut text = String::from("[packages]\n");
        for (name, requirement) in &self.packages {
            text.push_str(name);
            text.push_str(" = \"");
            text.push_str(requirement);
            text.push_str("\"\n");
        }
        text
    }

    pub fn package_count(&self) -> usize {
        self.packages.len()
    }
}

/// Imports installed packages into the requested configuration file.
/// An existing configuration is replaced only when force is selected.
pub fn import_configuration(
    path: &Path,
    installed: Vec<InstalledPackage>,
    policy: VersionPolicy,
    keep_importer: bool,
    force: bool,
) -> io::Result<usize> {
    if path.try_exists()? && !force {
        return Err(io::Error::new(io::ErrorKind::AlreadyExists, "configuration exists"));
    }
    let configuration = build_configuration(installed, policy, keep_importer)?;
    let serialized = configuration.render();
    fs::write(path, serialized)?;
    Ok(configuration.package_count())
}

pub fn read_configuration(path: &Path) -> io::Result<String> {
    let metadata = fs::metadata(path)?;
    if !metadata.is_file() {
        return Err(io::Error::new(io::ErrorKind::InvalidInput, "configuration is not a file"));
    }
    fs::read_to_string(path)
}
