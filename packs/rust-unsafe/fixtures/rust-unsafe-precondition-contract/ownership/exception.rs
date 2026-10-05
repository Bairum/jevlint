pub fn reclaim_packet() -> u8 {
    let mut packet = std::vec::Vec::<u8>::with_capacity(8);
    std::vec::Vec::extend_from_slice(&mut packet, &[2, 3, 5]);
    let original_capacity = std::vec::Vec::capacity(&packet);
    let initialized = std::vec::Vec::len(&packet);
    let unused_slots = original_capacity - initialized;
    let retained_capacity = original_capacity - unused_slots;
    let pointer = std::vec::Vec::as_ptr(&packet);
    let view = unsafe { std::slice::from_raw_parts(pointer, retained_capacity) };
    view.iter().copied().fold(0, u8::wrapping_add)
}
