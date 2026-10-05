use std::error::Error;
use std::fmt;
use std::io;
use std::net::SocketAddr;
use std::path::{Path, PathBuf};

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ServiceConfig {
    pub bind: SocketAddr,
    pub workers: usize,
    pub state_dir: PathBuf,
}

impl ServiceConfig {
    pub fn new(bind: SocketAddr, state_dir: PathBuf) -> Self {
        Self {
            bind,
            workers: 4,
            state_dir,
        }
    }

    pub fn render(&self) -> String {
        format!(
            "bind={}\nworkers={}\nstate_dir={}\n",
            self.bind,
            self.workers,
            self.state_dir.display()
        )
    }

    pub fn save(&self, path: &Path) -> io::Result<()> {
        std::fs::write(path, self.render())
    }
}

#[derive(Debug)]
pub enum ConfigError {
    FileNotFound(PathBuf),
    InvalidFormat { line: usize, message: String },
}

impl fmt::Display for ConfigError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::FileNotFound(path) => {
                write!(formatter, "configuration file not found: {}", path.display())
            }
            Self::InvalidFormat { line, message } => {
                write!(formatter, "configuration line {line}: {message}")
            }
        }
    }
}

impl Error for ConfigError {}

/// Loads a service configuration from disk. Invalid records include their
/// line number; filesystem failures are returned as configuration errors.
pub fn load_config(path: &Path) -> Result<ServiceConfig, ConfigError> {
    let content = std::fs::read_to_string(path)
        .map_err(|_| ConfigError::FileNotFound(path.to_owned()))?;
    parse_config(&content)
}

fn parse_config(content: &str) -> Result<ServiceConfig, ConfigError> {
    let mut bind = None;
    let mut workers = 4;
    let mut state_dir = None;

    for (index, record) in content.lines().enumerate() {
        let line = index + 1;
        let record = record.trim();
        if record.is_empty() || record.starts_with('#') {
            continue;
        }
        let (key, value) = record.split_once('=').ok_or_else(|| {
            invalid(line, "expected key=value")
        })?;
        let value = value.trim();
        match key.trim() {
            "bind" => {
                bind = Some(value.parse::<SocketAddr>().map_err(|cause| {
                    invalid(line, &cause.to_string())
                })?);
            }
            "workers" => {
                workers = value.parse::<usize>().map_err(|cause| {
                    invalid(line, &cause.to_string())
                })?;
                if workers == 0 || workers > 256 {
                    return Err(invalid(line, "workers must be between 1 and 256"));
                }
            }
            "state_dir" => {
                if value.is_empty() {
                    return Err(invalid(line, "state_dir must not be empty"));
                }
                state_dir = Some(PathBuf::from(value));
            }
            _ => return Err(invalid(line, "unknown configuration key")),
        }
    }

    Ok(ServiceConfig {
        bind: bind.ok_or_else(|| invalid(0, "bind is required"))?,
        workers,
        state_dir: state_dir.ok_or_else(|| invalid(0, "state_dir is required"))?,
    })
}

fn invalid(line: usize, message: &str) -> ConfigError {
    ConfigError::InvalidFormat {
        line,
        message: message.to_owned(),
    }
}
