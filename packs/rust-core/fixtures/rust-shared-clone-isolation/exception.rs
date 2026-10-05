/// Changes through either shared account handle are visible to all observers.
pub fn shared_balance() -> u64 {
    let original = std::rc::Rc::new(std::cell::RefCell::new(100u64));
    let observer = std::rc::Rc::clone(&original);
    *observer.borrow_mut() = 50;
    let balance = *original.borrow();
    balance
}
