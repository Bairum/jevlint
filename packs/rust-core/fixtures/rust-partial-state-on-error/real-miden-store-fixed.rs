#[derive(Clone, Default)]
pub struct NoteStore {
    pub expected_notes: std::collections::BTreeMap<u64, Vec<u8>>,
    pub note_scripts: std::collections::BTreeMap<u64, Vec<u8>>,
}

pub struct TransactionRequest {
    pub input_notes: Vec<(u64, Vec<u8>)>,
    pub output_scripts: Vec<(u64, Vec<u8>)>,
}

/// Execute using the request's output scripts. On execution failure, leave
/// all stored expected notes and note scripts unchanged for later requests.
pub fn execute_transaction(
    store: &mut NoteStore,
    request: TransactionRequest,
    execute: impl FnOnce(&std::collections::BTreeMap<u64, Vec<u8>>) -> std::io::Result<()>,
) -> std::io::Result<()> {
    let mut execution_scripts = store.note_scripts.clone();
    execution_scripts.extend(request.output_scripts);
    execute(&execution_scripts)?;
    store.expected_notes.extend(request.input_notes);
    store.note_scripts = execution_scripts;
    Ok(())
}
