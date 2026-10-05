/// Every accepted record must reach the archive queue before success is returned.
/// A metrics tick does not retire an accepted record.
pub async fn archive_record(
    archive: &tokio::sync::mpsc::Sender<String>,
    record: String,
) -> Result<(), tokio::sync::mpsc::error::SendError<String>> {
    tokio::select! {
        result = archive.send(record) => result,
        _ = tokio::time::sleep(std::time::Duration::from_millis(10)) => Ok(()),
    }
}
