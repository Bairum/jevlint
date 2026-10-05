use std::ffi::{c_char, CStr, CString};

extern "C" {
    fn foreign_alloc_name(id: u32) -> *mut c_char;
}

pub fn take_name(id: u32) -> Option<String> {
    // foreign_alloc_name returns null or an owned NUL-terminated string from the foreign
    // arena allocator. Non-null results must be released with foreign_release_name only;
    // they were not allocated by CString::into_raw or the Rust global allocator.
    let ptr = unsafe { foreign_alloc_name(id) };
    if ptr.is_null() {
        return None;
    }
    let text = unsafe { CStr::from_ptr(ptr) }.to_string_lossy().into_owned();
    let owner = unsafe { CString::from_raw(ptr) };
    drop(owner);
    Some(text)
}
