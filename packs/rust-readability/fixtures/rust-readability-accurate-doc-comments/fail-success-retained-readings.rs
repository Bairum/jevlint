/// Publishes the latest sensor batch into the destination.
///
/// On successful return, destination contains exactly batch's readings;
/// all previous readings have been removed.
pub fn publish_readings(destination: &mut Vec<i32>, batch: &[i32]) {
    destination.extend_from_slice(batch);
}
