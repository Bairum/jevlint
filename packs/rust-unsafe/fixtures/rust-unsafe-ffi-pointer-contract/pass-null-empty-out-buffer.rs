#[repr(C)]
pub struct Bytes {
    data: *const u8,
    len: usize,
}

extern "C" {
    fn get_bytes(id: u32, out: *mut Bytes) -> bool;
}

pub fn bytes(id: u32) -> Option<Vec<u8>> {
    // get_bytes initializes out on success. Empty buffers may use null; nonempty buffers
    // contain len initialized bytes in one allocation, with len <= isize::MAX, readable
    // and unmodified for this call. The library retains ownership and requires no release.
    let mut result = Bytes { data: std::ptr::null(), len: 0 };
    if !unsafe { get_bytes(id, &mut result) } {
        return None;
    }
    if result.len == 0 {
        return Some(Vec::new());
    }
    if result.data.is_null() {
        return None;
    }
    Some(unsafe { std::slice::from_raw_parts(result.data, result.len) }.to_vec())
}
