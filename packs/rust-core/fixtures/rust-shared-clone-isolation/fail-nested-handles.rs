/// Edits to a cloned worksheet must not affect cells in the original sheet.
/// Return the original first cell after setting the draft cell's value.
pub fn edit_worksheet() -> u32 {
    let original: Vec<std::rc::Rc<std::cell::Cell<u32>>> =
        vec![std::rc::Rc::new(std::cell::Cell::new(10))];
    let draft = original.clone();
    draft[0].set(20);
    original[0].get()
}
