pub fn inspect_first<F: FnOnce(&u8)>(callback: F) -> std::vec::Vec<std::boxed::Box<u8>> {
    // The API makes no resource-cleanup guarantee when callback unwinds.
    let mut values = std::vec::Vec::<std::boxed::Box<u8>>::new();
    std::vec::Vec::push(&mut values, std::boxed::Box::new(7_u8));
    let pointer = std::vec::Vec::as_mut_ptr(&mut values);
    unsafe { std::vec::Vec::set_len(&mut values, 0) };
    let value = std::mem::ManuallyDrop::new(unsafe { std::ptr::read(pointer) });
    callback(std::boxed::Box::as_ref(&value));
    std::vec::Vec::push(&mut values, std::mem::ManuallyDrop::into_inner(value));
    values
}
