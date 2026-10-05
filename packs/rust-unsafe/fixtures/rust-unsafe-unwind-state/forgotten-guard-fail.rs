struct Drain<'a> {
    values: &'a mut Vec<Box<u8>>,
}

impl Drop for Drain<'_> {
    fn drop(&mut self) {
        unsafe { self.values.set_len(0) };
    }
}

pub fn consume_single(values: &mut Vec<Box<u8>>) {
    if values.len() != 1 {
        return;
    }
    let guard = Drain { values };
    let item = unsafe { std::ptr::read(guard.values.as_ptr()) };
    drop(item);
    std::mem::forget(guard);
}
