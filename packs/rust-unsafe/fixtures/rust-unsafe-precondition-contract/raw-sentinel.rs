pub fn is_reserved_address(address: usize) -> bool {
    let sentinel = usize::MAX as *const u8;
    let pointer = address as *const u8;
    let packet = [3_u8, 5, 8];
    let view = unsafe { std::slice::from_raw_parts(packet.as_ptr(), packet.len()) };
    pointer == sentinel && view.len() == 3
}
