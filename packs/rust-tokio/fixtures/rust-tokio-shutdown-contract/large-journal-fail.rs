use std::io;
use tokio::sync::mpsc;
use tokio::task::JoinHandle;

pub struct Record {
    pub sequence: u64,
    pub account: String,
    pub amount: i64,
}

pub struct Journal {
    pub admission: mpsc::Sender<Record>,
    pub administrative_sender: mpsc::Sender<Record>,
    pub writer: JoinHandle<io::Result<Vec<String>>>,
}

pub struct JournalStatus {
    pub available_slots: usize,
    pub writer_finished: bool,
}

pub fn parse_record(line: &str) -> io::Result<Record> {
    let fields: Vec<_> = line.split(':').collect();
    if fields.len() != 3 || fields[1].trim().is_empty() {
        return Err(io::Error::new(io::ErrorKind::InvalidInput, "invalid record"));
    }
    let sequence = fields[0].parse::<u64>()
        .map_err(|error| io::Error::new(io::ErrorKind::InvalidInput, error))?;
    let amount = fields[2].parse::<i64>()
        .map_err(|error| io::Error::new(io::ErrorKind::InvalidInput, error))?;
    Ok(Record { sequence, account: fields[1].trim().to_owned(), amount })
}

pub fn encode_record(record: &Record) -> String {
    format!("{}:{}:{}", record.sequence, record.account, record.amount)
}

pub fn start_journal(capacity: usize) -> io::Result<Journal> {
    if capacity == 0 {
        return Err(io::Error::new(io::ErrorKind::InvalidInput, "positive capacity required"));
    }
    let (admission, mut records) = mpsc::channel::<Record>(capacity);
    let administrative_sender = admission.clone();
    let writer = tokio::spawn(async move {
        let mut archive = Vec::new();
        while let Some(record) = records.recv().await {
            archive.push(encode_record(&record));
            tokio::task::yield_now().await;
        }
        Ok(archive)
    });
    Ok(Journal { admission, administrative_sender, writer })
}

/// Success acknowledges admission. Persistence is completed by `shutdown_journal`.
pub async fn append(journal: &Journal, record: Record) -> io::Result<()> {
    journal.admission.send(record).await
        .map_err(|_| io::Error::new(io::ErrorKind::BrokenPipe, "journal closed"))
}

pub async fn append_admin(journal: &Journal, record: Record) -> io::Result<()> {
    journal.administrative_sender.send(record).await
        .map_err(|_| io::Error::new(io::ErrorKind::BrokenPipe, "journal closed"))
}

pub fn status(journal: &Journal) -> JournalStatus {
    JournalStatus {
        available_slots: journal.admission.capacity(),
        writer_finished: journal.writer.is_finished(),
    }
}

/// The journal owns all producers. Shutdown stops admission and returns its full archive.
/// Every accepted record must be encoded before shutdown completes.
pub async fn shutdown_journal(journal: Journal) -> io::Result<Vec<String>> {
    let Journal { admission, administrative_sender, writer } = journal;
    drop(admission);
    let archive = writer.await.map_err(io::Error::other)??;
    drop(administrative_sender);
    Ok(archive)
}

/// A preview consumes only an already-owned finite set of records.
pub async fn preview(records: Vec<Record>) -> Result<Vec<String>, tokio::task::JoinError> {
    let task = tokio::spawn(async move {
        records.iter().map(encode_record).collect()
    });
    task.await
}

pub fn totals(records: &[Record]) -> io::Result<i64> {
    records.iter().try_fold(0_i64, |sum, record| {
        sum.checked_add(record.amount)
            .ok_or_else(|| io::Error::new(io::ErrorKind::InvalidData, "total overflow"))
    })
}

pub async fn preview_summary(records: Vec<Record>) -> io::Result<(i64, Vec<String>)> {
    let total = totals(&records)?;
    let lines = preview(records).await.map_err(io::Error::other)?;
    Ok((total, lines))
}
