pub fn preview_balance() -> u64 {
    let original = std::rc::Rc::new(std::cell::RefCell::new(100u64));
    let snapshot = std::rc::Rc::clone(&original);
    *snapshot.borrow_mut() = 50;
    let balance = *original.borrow();
    balance
}
