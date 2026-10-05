/// Identical entries share a counter; edits are visible through every entry.
pub fn repeated_counter() -> u32 {
    let counter = std::rc::Rc::new(std::cell::Cell::new(0));
    let entries = vec![std::rc::Rc::clone(&counter), std::rc::Rc::clone(&counter)];
    entries[0].set(3);
    entries[1].get()
}
