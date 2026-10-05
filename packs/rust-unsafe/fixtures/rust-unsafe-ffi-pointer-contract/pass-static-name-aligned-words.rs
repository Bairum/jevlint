use std::ffi::{c_char, CStr};

#[repr(C)]
pub struct Entry {
    name: *const c_char,
    words: *const u32,
    count: usize,
}

extern "C" {
    fn get_entry(id: u32, out: *mut Entry) -> bool;
}

pub fn entry(id: u32) -> Option<(String, Vec<u32>)> {
    // get_entry initializes every field on success. name is non-null and NUL-terminated.
    // words is non-null, aligned for u32, and points to count initialized values within
    // one allocation whose size is <= isize::MAX. Both allocations are immutable and
    // remain live for the process lifetime. The library owns them; callers must not free.
    let mut result = Entry {
        name: std::ptr::null(),
        words: std::ptr::null(),
        count: 0,
    };
    if !unsafe { get_entry(id, &mut result) } {
        return None;
    }
    let name = unsafe { CStr::from_ptr(result.name) }.to_string_lossy().into_owned();
    let words = unsafe { std::slice::from_raw_parts(result.words, result.count) }.to_vec();
    Some((name, words))
}
