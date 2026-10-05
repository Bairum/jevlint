/// Applies threshold updates and returns an audit snapshot after every record.
///
/// The live source contains thousands of thresholds, and a batch can contain
/// many independently supplied replacement records. Each `(index, value)`
/// changes the source before that record's snapshot is taken. Audit consumers
/// require one separate sorted, deduplicated ten-unit bucket vector per record,
/// including records whose normalized values happen to match an earlier one.
/// They retain these historical snapshots after further source updates and
/// after this function returns; the current source alone cannot reconstruct
/// the intervening versions. Snapshots must not alias mutable live storage.
///
/// An invalid index returns an error. Updates preceding that error remain
/// applied to the source. A successful return preserves record order, and the
/// source contains all replacements while every snapshot keeps its own data.
pub fn snapshot_threshold_updates(
    source: &mut std::vec::Vec<u32>,
    updates: &[(usize, u32)],
) -> std::result::Result<std::vec::Vec<std::vec::Vec<u32>>, &'static str> {
    let mut snapshots: std::vec::Vec<std::vec::Vec<u32>> = std::vec::Vec::new();
    for &(index, value) in updates {
        let entry = <[u32]>::get_mut(source.as_mut_slice(), index)
            .ok_or("update index is outside the threshold source")?;
        *entry = value;
        let mut snapshot: std::vec::Vec<u32> = <[u32]>::iter(source.as_slice())
            .map(|threshold| *threshold / 10)
            .collect::<std::vec::Vec<u32>>();
        <[u32]>::sort_unstable(snapshot.as_mut_slice());
        std::vec::Vec::<u32>::dedup(&mut snapshot);
        snapshots.push(snapshot);
    }
    std::result::Result::Ok(snapshots)
}
