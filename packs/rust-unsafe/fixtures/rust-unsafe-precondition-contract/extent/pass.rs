pub fn tail_checksum() -> u16 {
    let packet = [11_u16, 22, 33, 44];
    let skipped_elements = 2;
    let tail_elements = packet.len() - skipped_elements;
    let start = unsafe { packet.as_ptr().add(skipped_elements) };
    let tail = unsafe { std::slice::from_raw_parts::<u16>(start, tail_elements) };
    tail.iter().copied().fold(0, u16::wrapping_add)
}
