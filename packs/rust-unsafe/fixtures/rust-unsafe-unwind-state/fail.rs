pub fn inspect_first<F: FnOnce(&u8)>(callback: F) -> std::vec::Vec<std::boxed::Box<u8>> {
    let mut values = std::vec::Vec::<std::boxed::Box<u8>>::new();
    std::vec::Vec::push(&mut values, std::boxed::Box::new(7_u8));
    let pointer = std::vec::Vec::as_mut_ptr(&mut values);
    let value = unsafe { std::ptr::read(pointer) };
    callback(std::boxed::Box::as_ref(&value));
    unsafe { std::ptr::write(pointer, value) };
    values
}
