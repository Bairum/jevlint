#[repr(C)]
#[derive(Clone, Copy)]
pub struct Header {
    pub kind: u8,
    pub sequence: u32,
}

pub fn encode_headers(headers: Vec<Header>) -> Vec<u8> {
    let length = headers.len().checked_mul(5).expect("packet too large");
    let mut output = Vec::<u8>::with_capacity(length);
    for header in headers {
        let sequence = header.sequence.to_le_bytes();
        let bytes = [header.kind, sequence[0], sequence[1], sequence[2], sequence[3]];
        let offset = output.len();
        unsafe {
            std::ptr::copy_nonoverlapping(bytes.as_ptr(), output.as_mut_ptr().add(offset), bytes.len());
            output.set_len(offset + bytes.len());
        }
    }
    output
}
