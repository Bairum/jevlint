#[repr(C)]
pub struct Header {
    pub kind: u8,
    pub sequence: u32,
}

pub fn header(kind: u8, sequence: u32) -> Header {
    let mut storage = std::mem::MaybeUninit::<Header>::uninit();
    let pointer = storage.as_mut_ptr();
    unsafe {
        std::ptr::addr_of_mut!((*pointer).kind).write(kind);
        std::ptr::addr_of_mut!((*pointer).sequence).write(sequence);
        storage.assume_init()
    }
}
