pub fn clone_batch(source: &[u8], mut clone_item: impl FnMut(&u8) -> Box<u8>) -> Vec<Box<u8>> {
    let mut output = Vec::<Box<u8>>::with_capacity(source.len());
    let pointer = output.as_mut_ptr();
    for (index, item) in source.iter().enumerate() {
        let cloned = clone_item(item);
        unsafe {
            std::ptr::write(pointer.add(index), cloned);
            output.set_len(index + 1);
        }
    }
    output
}
