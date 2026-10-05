// Queue capacity policy includes allocator and index bookkeeping per record.
// This estimate can be retuned without changing payload encoding.
const PER_RECORD_ACCOUNTING_OVERHEAD_BYTES: usize = 48;

pub fn batch_fits_budget(payload_lengths: &[usize], available_bytes: usize) -> bool {
    let accounted_bytes = payload_lengths.iter().try_fold(0usize, |used, &payload| {
        payload
            .checked_add(PER_RECORD_ACCOUNTING_OVERHEAD_BYTES)
            .and_then(|record| used.checked_add(record))
    });
    matches!(accounted_bytes, Some(used) if used <= available_bytes)
}
