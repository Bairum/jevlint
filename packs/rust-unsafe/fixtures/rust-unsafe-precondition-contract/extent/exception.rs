pub fn tail_checksum() -> u8 {
    let packet = [11_u16, 22, 33, 44];
    let skipped_elements = 2;
    let tail_elements = packet.len() - skipped_elements;
    let tail_bytes = tail_elements * std::mem::size_of::<u16>();
    let start = unsafe { packet.as_ptr().add(skipped_elements) }.cast::<u8>();
    let tail = unsafe { std::slice::from_raw_parts::<u8>(start, tail_bytes) };
    tail.iter().copied().fold(0, u8::wrapping_add)
}
