#[derive(Clone, Debug)]
pub struct ExportRow {
    pub account: String,
    pub cents: i64,
}

pub struct ExportPlan {
    pub directory: std::path::PathBuf,
    pub batch: String,
    pub rows: Vec<ExportRow>,
}

impl ExportPlan {
    pub fn new(directory: std::path::PathBuf, batch: String) -> Self {
        Self {
            directory,
            batch,
            rows: Vec::new(),
        }
    }

    pub fn add(&mut self, account: String, cents: i64) {
        self.rows.push(ExportRow { account, cents });
    }

    pub fn total(&self) -> i64 {
        self.rows.iter().map(|row| row.cents).sum()
    }

    pub fn destination(&self) -> std::path::PathBuf {
        self.directory.join(format!("{}.csv", self.batch))
    }
}

pub fn validate_batch(batch: &str) -> std::io::Result<()> {
    if batch.is_empty() || !batch.bytes().all(|byte| byte.is_ascii_alphanumeric()) {
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "batch must be alphanumeric",
        ));
    }
    Ok(())
}

pub fn encode_rows(rows: &[ExportRow]) -> Vec<u8> {
    let mut text = String::from("account,cents\n");
    for row in rows {
        let account = row.account.replace(',', "_").replace('\n', " ");
        text.push_str(&format!("{},{}\n", account, row.cents));
    }
    text.into_bytes()
}

/// Save a new batch without replacing another export. Multiple processes
/// can publish the same batch in this shared directory concurrently.
/// On success all bytes have been submitted to the file; no durability promise.
pub fn publish_batch(plan: &ExportPlan) -> std::io::Result<usize> {
    validate_batch(&plan.batch)?;
    let path = plan.destination();
    let bytes = encode_rows(&plan.rows);
    if path.exists() {
        return Err(std::io::Error::new(
            std::io::ErrorKind::AlreadyExists,
            "batch already exported",
        ));
    }
    let mut file = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(&path)?;
    std::io::Write::write_all(&mut file, &bytes)?;
    Ok(bytes.len())
}

/// Replace the dashboard's most recent summary after each completed batch.
pub fn replace_summary(directory: &std::path::Path, plan: &ExportPlan) -> std::io::Result<()> {
    let path = directory.join("latest.txt");
    let content = format!("batch={}\nrows={}\ntotal={}\n", plan.batch, plan.rows.len(), plan.total());
    std::fs::write(path, content)
}

/// Reserve a new audit entry without replacing any existing entry.
pub fn reserve_audit(path: &std::path::Path) -> std::io::Result<std::fs::File> {
    std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(path)
}

pub fn load_summary(directory: &std::path::Path) -> std::io::Result<String> {
    std::fs::read_to_string(directory.join("latest.txt"))
}

pub fn batch_is_empty(plan: &ExportPlan) -> bool {
    plan.rows.is_empty()
}
