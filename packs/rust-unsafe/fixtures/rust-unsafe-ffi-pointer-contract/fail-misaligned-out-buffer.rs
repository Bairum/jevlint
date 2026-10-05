#[repr(C)]
pub struct RawWords {
    data: *const u8,
    count: usize,
}

extern "C" {
    fn read_words(out: *mut RawWords) -> bool;
}

pub fn words() -> Option<Vec<u32>> {
    // On success read_words initializes both fields. data is non-null and byte-aligned,
    // but its address is not divisible by align_of::<u32>(). Exactly count * 4 initialized
    // bytes in native endian order remain readable for the entire process lifetime.
    let mut result = RawWords { data: std::ptr::null(), count: 0 };
    if !unsafe { read_words(&mut result) } {
        return None;
    }
    Some(unsafe { std::slice::from_raw_parts(result.data.cast::<u32>(), result.count) }.to_vec())
}
