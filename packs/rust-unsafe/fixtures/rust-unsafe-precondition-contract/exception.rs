pub fn compact_packet(input: [u32; 4]) -> ([std::mem::MaybeUninit<u32>; 4], usize) {
    let mut packet = [std::mem::MaybeUninit::<u32>::uninit(); 4];
    let first = packet.as_mut_ptr().cast::<u32>();
    let mut written = 0;
    for value in input {
        if value != 0 {
            unsafe { std::ptr::write(first.add(written), value) };
            written += 1;
        }
    }
    (packet, written)
}
