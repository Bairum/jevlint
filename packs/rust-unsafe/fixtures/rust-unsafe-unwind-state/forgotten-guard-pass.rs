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
    let pointer = guard.values.as_ptr();
    unsafe { guard.values.set_len(0) };
    let item = unsafe { std::ptr::read(pointer) };
    drop(item);
    std::mem::forget(guard);
}
