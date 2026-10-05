/// Publishes the latest sensor batch into the destination.
///
/// On successful return, batch's readings have been appended in order;
/// all previous readings remain before the appended batch.
pub fn publish_readings(destination: &mut Vec<i32>, batch: &[i32]) {
    destination.extend_from_slice(batch);
}
