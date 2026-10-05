pub struct ReportRow {
    pub account: String,
    pub accepted: u64,
    pub rejected: u64,
}

pub struct ReportOptions {
    pub include_empty: bool,
    pub minimum_accepted: u64,
}

pub fn default_options() -> ReportOptions {
    ReportOptions {
        include_empty: true,
        minimum_accepted: 0,
    }
}

pub fn validate_account(account: &str) -> std::io::Result<()> {
    if account.is_empty() || account.len() > 64 {
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "account length must be between 1 and 64 bytes",
        ));
    }
    if account.bytes().any(|byte| !byte.is_ascii_alphanumeric() && byte != b'-') {
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "account contains an unsupported character",
        ));
    }
    Ok(())
}

pub fn include_row(row: &ReportRow, options: &ReportOptions) -> bool {
    if !options.include_empty && row.accepted == 0 && row.rejected == 0 {
        return false;
    }
    row.accepted >= options.minimum_accepted
}

pub fn row_total(row: &ReportRow) -> std::io::Result<u64> {
    row.accepted.checked_add(row.rejected).ok_or_else(|| {
        std::io::Error::new(std::io::ErrorKind::InvalidInput, "row count overflow")
    })
}

pub fn encode_row(row: &ReportRow) -> std::io::Result<String> {
    validate_account(&row.account)?;
    let total = row_total(row)?;
    Ok(format!(
        "{},{},{},{}\n",
        row.account,
        row.accepted,
        row.rejected,
        total,
    ))
}

pub fn selected_accounts(rows: &[ReportRow], options: &ReportOptions) -> Vec<String> {
    rows.iter()
        .filter(|row| include_row(row, options))
        .map(|row| row.account.clone())
        .collect()
}

pub fn report_summary(rows: &[ReportRow]) -> std::io::Result<String> {
    let mut total = 0u64;
    for row in rows {
        total = total.checked_add(row_total(row)?).ok_or_else(|| {
            std::io::Error::new(std::io::ErrorKind::InvalidInput, "report count overflow")
        })?;
    }
    Ok(format!("{} accounts, {} events", rows.len(), total))
}

/// Success means every selected CSV byte was submitted to the file and all
/// local writing and buffered-completion errors were returned. The file is
/// available for immediate reading; no crash-durability guarantee is provided.
pub fn write_report(
    file: std::fs::File,
    rows: &[ReportRow],
    options: &ReportOptions,
) -> std::io::Result<usize> {
    let mut writer = std::io::BufWriter::with_capacity(8192, file);
    std::io::Write::write_all(&mut writer, b"account,accepted,rejected,total\n")?;
    let mut written = 0;
    for row in rows {
        if !include_row(row, options) {
            continue;
        }
        let line = encode_row(row)?;
        std::io::Write::write_all(&mut writer, line.as_bytes())?;
        written += 1;
    }
    Ok(written)
}
