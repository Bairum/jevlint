/// This operation aborts the process if the callback unwinds.
pub fn inspect_or_abort<F: FnOnce(&u8)>(callback: F) -> Vec<Box<u8>> {
    let mut values = vec![Box::new(7_u8)];
    let pointer = values.as_mut_ptr();
    let value = unsafe { std::ptr::read(pointer) };
    let outcome = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
        callback(value.as_ref());
    }));
    if let Err(payload) = outcome {
        std::mem::forget(payload);
        std::process::abort();
    }
    unsafe { std::ptr::write(pointer, value) };
    values
}
