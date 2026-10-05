/// Each observer receives the same immutable vocabulary in a shared owner.
pub fn vocabulary() -> (std::sync::Arc<Vec<String>>, std::sync::Arc<Vec<String>>) {
    let words = std::sync::Arc::new(vec![String::from("north"), String::from("south")]);
    let observer = std::sync::Arc::clone(&words);
    (words, observer)
}
