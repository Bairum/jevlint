/// Counts measurements landing in a configured ten-unit threshold bucket.
///
/// The ingestion workload normally contains at least 100,000 measurements and
/// thousands of threshold entries. Thresholds are immutable throughout this
/// batch. Normalization divides each threshold by ten, orders the buckets, and
/// removes duplicates. Each measurement contributes either zero or one to the
/// count, regardless of the number of thresholds in its bucket.
///
/// The normalized `std::vec::Vec<u32>` is only borrowed for binary search; no
/// measurement consumer mutates it, retains it, or requires an owned snapshot.
/// The returned scalar is the only result retained by the ingestion caller.
pub fn count_threshold_measurements(thresholds: &[u32], measurements: &[u32]) -> usize {
    let mut matches = 0;
    for measurement in measurements {
        let mut normalized: std::vec::Vec<u32> = <[u32]>::iter(thresholds)
            .map(|threshold| *threshold / 10)
            .collect::<std::vec::Vec<u32>>();
        <[u32]>::sort_unstable(normalized.as_mut_slice());
        std::vec::Vec::<u32>::dedup(&mut normalized);
        if <[u32]>::binary_search(normalized.as_slice(), &(*measurement / 10)).is_ok() {
            matches += 1;
        }
    }
    matches
}
