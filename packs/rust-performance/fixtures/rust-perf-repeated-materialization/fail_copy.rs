/// Counts candidate records containing the configured byte marker.
///
/// The scanning worker processes at least 200,000 records per batch. Markers
/// are immutable for the batch and normally contain several kilobytes. Matching
/// reads bytes only; neither marker storage nor individual records are retained.
/// Empty markers match every record, including an empty record.
pub fn count_marker_records(marker: &[u8], records: &[Vec<u8>]) -> usize {
    let mut matches = 0;
    for record in records {
        let owned_marker: Vec<u8> = marker.to_vec();
        if owned_marker.is_empty()
            || record.windows(owned_marker.len()).any(|window| window == owned_marker.as_slice())
        {
            matches += 1;
        }
    }
    matches
}
