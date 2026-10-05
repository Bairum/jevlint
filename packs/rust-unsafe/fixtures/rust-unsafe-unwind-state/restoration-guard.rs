pub struct Restore<'a> {
    values: &'a mut Vec<Box<u8>>,
    item: Option<Box<u8>>,
}

impl Drop for Restore<'_> {
    fn drop(&mut self) {
        if let Some(item) = self.item.take() {
            self.values.push(item);
        }
    }
}

pub fn inspect_with_guard<F: FnOnce(&u8)>(values: &mut Vec<Box<u8>>, callback: F) {
    if values.len() != 1 {
        return;
    }
    let pointer = values.as_ptr();
    unsafe { values.set_len(0) };
    let item = unsafe { std::ptr::read(pointer) };
    let guard = Restore { values, item: Some(item) };
    callback(guard.item.as_ref().unwrap().as_ref());
}

pub fn forget_restoration(values: &mut Vec<Box<u8>>) {
    if values.len() != 1 {
        return;
    }
    let pointer = values.as_ptr();
    unsafe { values.set_len(0) };
    let item = unsafe { std::ptr::read(pointer) };
    let guard = Restore { values, item: Some(item) };
    std::mem::forget(guard);
}
