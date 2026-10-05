#[derive(Debug)]
pub struct Project {
    pub name: String,
    pub workers: usize,
    pub root: std::path::PathBuf,
}

#[derive(Debug)]
pub enum LoadFailure {
    Read {
        path: std::path::PathBuf,
        cause: std::io::Error,
    },
    Syntax {
        line: usize,
        message: String,
    },
}

impl std::fmt::Display for LoadFailure {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Read { path, cause } => {
                write!(formatter, "{}: {}", path.display(), cause)
            }
            Self::Syntax { line, message } => {
                write!(formatter, "manifest line {}: {}", line, message)
            }
        }
    }
}

impl std::error::Error for LoadFailure {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::Read { cause, .. } => Some(cause),
            Self::Syntax { .. } => None,
        }
    }
}

/// Read the project's administrator-managed manifest. Return Read with the
/// exact path and original OS cause so callers offer creation for NotFound
/// and permission repair for PermissionDenied. Syntax is reserved for parsing.
pub fn read_manifest(path: &std::path::Path) -> Result<String, LoadFailure> {
    std::fs::read_to_string(path).map_err(|_| LoadFailure::Syntax {
        line: 0,
        message: "manifest unavailable".to_owned(),
    })
}

pub fn manifest_path(root: &std::path::Path) -> std::path::PathBuf {
    root.join("project.conf")
}

/// Parse the two manifest settings; errors identify the relevant text line.
pub fn parse_manifest(text: &str, root: &std::path::Path) -> Result<Project, LoadFailure> {
    let mut name = None;
    let mut workers = 1;
    for (index, raw) in text.lines().enumerate() {
        let line = raw.trim();
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let (key, value) = line.split_once('=').ok_or_else(|| LoadFailure::Syntax {
            line: index + 1,
            message: "expected key=value".to_owned(),
        })?;
        match key.trim() {
            "name" => name = Some(value.trim().to_owned()),
            "workers" => {
                workers = value.trim().parse::<usize>().map_err(|cause| LoadFailure::Syntax {
                    line: index + 1,
                    message: format!("workers: {}", cause),
                })?;
            }
            _ => return Err(LoadFailure::Syntax {
                line: index + 1,
                message: format!("unknown setting {}", key),
            }),
        }
    }
    let name = name.filter(|value| !value.is_empty()).ok_or_else(|| LoadFailure::Syntax {
        line: 0,
        message: "name is required".to_owned(),
    })?;
    Ok(Project { name, workers, root: root.to_path_buf() })
}

pub fn load_project(root: &std::path::Path) -> Result<Project, LoadFailure> {
    let text = read_manifest(&manifest_path(root))?;
    parse_manifest(&text, root)
}

/// Read an optional local policy; missing files select the built-in policy.
/// Other I/O failures retain their original identity.
pub fn read_policy(root: &std::path::Path) -> std::io::Result<String> {
    match std::fs::read_to_string(root.join("policy.conf")) {
        Ok(text) => Ok(text),
        Err(cause) if cause.kind() == std::io::ErrorKind::NotFound => {
            Ok("allow-local=true".to_owned())
        }
        Err(cause) => Err(cause),
    }
}

pub fn worker_label(project: &Project, index: usize) -> String {
    format!("{}-{}", project.name, index)
}
