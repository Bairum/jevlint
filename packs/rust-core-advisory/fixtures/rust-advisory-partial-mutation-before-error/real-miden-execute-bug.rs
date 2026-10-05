use std::collections::BTreeMap;
use std::fmt;

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum NoteStatus {
    Expected,
    Consumed,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Note {
    pub id: u64,
    pub owner: u64,
    pub amount: u64,
    pub status: NoteStatus,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Store {
    pub accounts: BTreeMap<u64, u64>,
    pub notes: BTreeMap<u64, Note>,
    pub scripts: BTreeMap<u64, String>,
    pub transactions: BTreeMap<u64, Receipt>,
}

#[derive(Clone, Debug)]
pub struct Request {
    pub id: u64,
    pub account: u64,
    pub debit: u64,
    pub inputs: Vec<Note>,
    pub output_scripts: BTreeMap<u64, String>,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Receipt {
    pub transaction: u64,
    pub account: u64,
    pub remaining_balance: u64,
    pub consumed: Vec<u64>,
}

#[derive(Debug, PartialEq, Eq)]
pub enum ExecutionError {
    DuplicateTransaction,
    UnknownAccount,
    InsufficientBalance,
    ForeignNote(u64),
    MissingScript(u64),
    RejectedScript(u64),
}

impl fmt::Display for ExecutionError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(formatter, "transaction execution failed: {self:?}")
    }
}

impl std::error::Error for ExecutionError {}

impl Store {
    pub fn empty() -> Self {
        Self {
            accounts: BTreeMap::new(),
            notes: BTreeMap::new(),
            scripts: BTreeMap::new(),
            transactions: BTreeMap::new(),
        }
    }

    pub fn expected_notes(&self, account: u64) -> Vec<u64> {
        self.notes
            .values()
            .filter(|note| note.owner == account && note.status == NoteStatus::Expected)
            .map(|note| note.id)
            .collect()
    }
}

pub fn validate_request(store: &Store, request: &Request) -> Result<(), ExecutionError> {
    if store.transactions.contains_key(&request.id) {
        return Err(ExecutionError::DuplicateTransaction);
    }
    for note in &request.inputs {
        if note.owner != request.account {
            return Err(ExecutionError::ForeignNote(note.id));
        }
    }
    Ok(())
}

fn run_executor(
    accounts: &BTreeMap<u64, u64>,
    scripts: &BTreeMap<u64, String>,
    request: &Request,
) -> Result<Receipt, ExecutionError> {
    let balance = accounts
        .get(&request.account)
        .ok_or(ExecutionError::UnknownAccount)?;
    let remaining_balance = balance
        .checked_sub(request.debit)
        .ok_or(ExecutionError::InsufficientBalance)?;
    for script_id in request.output_scripts.keys() {
        let script = scripts
            .get(script_id)
            .ok_or(ExecutionError::MissingScript(*script_id))?;
        if script != "accept" {
            return Err(ExecutionError::RejectedScript(*script_id));
        }
    }
    Ok(Receipt {
        transaction: request.id,
        account: request.account,
        remaining_balance,
        consumed: request.inputs.iter().map(|note| note.id).collect(),
    })
}

pub fn execute_transaction(
    store: &mut Store,
    request: &Request,
) -> Result<Receipt, ExecutionError> {
    validate_request(store, request)?;
    for note in &request.inputs {
        let mut expected = note.clone();
        expected.status = NoteStatus::Expected;
        store.notes.insert(note.id, expected);
    }
    store.scripts.extend(request.output_scripts.clone());
    let receipt = run_executor(&store.accounts, &store.scripts, request)?;
    store.accounts.insert(receipt.account, receipt.remaining_balance);
    for note_id in &receipt.consumed {
        if let Some(note) = store.notes.get_mut(note_id) {
            note.status = NoteStatus::Consumed;
        }
    }
    store.transactions.insert(receipt.transaction, receipt.clone());
    Ok(receipt)
}

pub fn transaction_summary(receipt: &Receipt) -> String {
    format!(
        "transaction={} account={} remaining={} inputs={}",
        receipt.transaction,
        receipt.account,
        receipt.remaining_balance,
        receipt.consumed.len(),
    )
}
