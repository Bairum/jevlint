pub struct ActiveLease<'a> {
    pub active: &'a std::cell::Cell<bool>,
}

impl std::ops::Drop for ActiveLease<'_> {
    /// Clear the local activation flag on every exit, including unwind.
    fn drop(&mut self) {
        self.active.set(false);
    }
}
