/// Every accepted record must reach the archive queue before success is returned.
pub async fn archive_record(
    archive: &tokio::sync::mpsc::Sender<String>,
    record: String,
) -> Result<(), tokio::sync::mpsc::error::SendError<String>> {
    loop {
        tokio::select! {
            result = archive.reserve() => {
                match result {
                    Ok(permit) => {
                        permit.send(record);
                        return Ok(());
                    }
                    Err(_) => return Err(tokio::sync::mpsc::error::SendError(record)),
                }
            }
            _ = tokio::time::sleep(std::time::Duration::from_millis(10)) => {}
        }
    }
}
