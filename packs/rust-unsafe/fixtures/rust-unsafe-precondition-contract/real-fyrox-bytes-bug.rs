#[repr(C)]
#[derive(Clone, Copy)]
pub struct Header {
    pub kind: u8,
    pub sequence: u32,
}

pub fn encode_headers(headers: Vec<Header>) -> Vec<u8> {
    let mut storage = std::mem::ManuallyDrop::new(headers);
    let length = storage.len() * std::mem::size_of::<Header>();
    let capacity = storage.capacity() * std::mem::size_of::<Header>();
    let pointer = storage.as_mut_ptr().cast::<u8>();
    unsafe { Vec::from_raw_parts(pointer, length, capacity) }
}
