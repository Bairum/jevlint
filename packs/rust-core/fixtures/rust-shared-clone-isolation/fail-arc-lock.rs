/// Produce a revised draft without changing the live settings.
/// Return both the live settings and the revised draft for display.
pub fn revise_settings() -> (Vec<String>, Vec<String>) {
    let original = std::sync::Arc::new(std::sync::Mutex::new(vec![String::from("normal")]));
    let draft = original.clone();
    draft.lock().unwrap().push(String::from("preview"));
    let live = original.lock().unwrap().clone();
    let revised = draft.lock().unwrap().clone();
    (live, revised)
}
