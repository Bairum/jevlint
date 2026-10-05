pub fn batch_fits_budget(payload_lengths: &[usize], available_bytes: usize) -> bool {
    let accounted_bytes = payload_lengths.iter().try_fold(0usize, |used, &payload| {
        payload.checked_add(48).and_then(|record| used.checked_add(record))
    });
    matches!(accounted_bytes, Some(used) if used <= available_bytes)
}
