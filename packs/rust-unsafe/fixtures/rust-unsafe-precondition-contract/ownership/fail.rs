pub fn reclaim_packet() -> std::vec::Vec<u8> {
    let mut packet = std::vec::Vec::<u8>::with_capacity(8);
    std::vec::Vec::extend_from_slice(&mut packet, &[2, 3, 5]);
    let mut storage = std::mem::ManuallyDrop::new(packet);
    let original_capacity = std::vec::Vec::capacity(&storage);
    let initialized = std::vec::Vec::len(&storage);
    let unused_slots = original_capacity - initialized;
    let retained_capacity = original_capacity - unused_slots;
    let pointer = std::vec::Vec::as_mut_ptr(&mut storage);
    unsafe { std::vec::Vec::from_raw_parts(pointer, initialized, retained_capacity) }
}
