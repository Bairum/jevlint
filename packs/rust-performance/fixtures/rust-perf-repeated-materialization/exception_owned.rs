/// Queues private working vectors and retains each consumer's filtered result.
///
/// Job batches repeatedly use the same immutable source, typically thousands
/// of entries. Each cutoff belongs to an independent job. The queue's payload
/// contract is `(u32, std::vec::Vec<u32>)`: jobs own the complete input before
/// the receiver applies its cutoff. Each consumer mutates and retains its own
/// result independently. The returned vectors must remain valid after the
/// caller changes or drops the source storage.
///
/// This function keeps both channel endpoints local. All jobs are sent before
/// consumption, and closing the sender lets the receiver finish after draining
/// the queue. Results retain cutoff order, source order, and duplicate values.
/// The caller bounds batch size to accommodate queued working vectors.
pub fn retain_owned_job_results(source: &[u32], cutoffs: &[u32]) -> std::vec::Vec<std::vec::Vec<u32>> {
    let (sender, receiver): (
        std::sync::mpsc::Sender<(u32, std::vec::Vec<u32>)>,
        std::sync::mpsc::Receiver<(u32, std::vec::Vec<u32>)>,
    ) = std::sync::mpsc::channel::<(u32, std::vec::Vec<u32>)>();
    for &cutoff in cutoffs {
        let working_values: std::vec::Vec<u32> = <[u32]>::iter(source)
            .copied()
            .collect::<std::vec::Vec<u32>>();
        std::sync::mpsc::Sender::send(&sender, (cutoff, working_values))
            .expect("the receiver remains alive until all job payloads have been sent");
    }
    std::mem::drop(sender);
    let mut retained: std::vec::Vec<std::vec::Vec<u32>> = std::vec::Vec::new();
    for (cutoff, mut working_values) in receiver {
        std::vec::Vec::<u32>::retain(&mut working_values, |value| *value >= cutoff);
        retained.push(working_values);
    }
    retained
}
