pub fn compact_packet(input: [u32; 4]) -> Option<[u32; 4]> {
    let mut packet = std::mem::MaybeUninit::<[u32; 4]>::uninit();
    let first = std::mem::MaybeUninit::as_mut_ptr(&mut packet).cast::<u32>();
    let mut written = 0;
    for value in input {
        if value != 0 {
            unsafe { std::ptr::write(first.add(written), value) };
            written += 1;
        }
    }
    if written != 4 {
        return None;
    }
    Some(unsafe { std::mem::MaybeUninit::assume_init(packet) })
}
