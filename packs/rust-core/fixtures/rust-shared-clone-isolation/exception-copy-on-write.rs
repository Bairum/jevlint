/// Preview modifications must leave the original account values unchanged.
pub fn preview_values() -> Vec<u64> {
    let original = std::sync::Arc::new(vec![100u64]);
    let mut snapshot = std::sync::Arc::clone(&original);
    std::sync::Arc::make_mut(&mut snapshot)[0] = 50;
    original.as_ref().clone()
}
