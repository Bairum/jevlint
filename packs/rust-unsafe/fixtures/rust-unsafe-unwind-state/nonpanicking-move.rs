pub fn rotate_first(mut values: Vec<Box<u8>>) -> Vec<Box<u8>> {
    if values.len() < 2 {
        return values;
    }
    let pointer = values.as_mut_ptr();
    unsafe {
        let first = std::ptr::read(pointer);
        let second = std::ptr::read(pointer.add(1));
        std::ptr::write(pointer, second);
        std::ptr::write(pointer.add(1), first);
    }
    values
}
