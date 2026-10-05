use std::ffi::{c_char, CStr};

extern "C" {
    fn foreign_alloc_name(id: u32) -> *mut c_char;
    fn foreign_release_name(name: *mut c_char);
}

pub fn take_name(id: u32) -> Option<String> {
    // foreign_alloc_name returns null or an owned NUL-terminated string from the foreign
    // arena allocator. The string remains readable until foreign_release_name is called.
    // Every non-null result must be passed exactly once to foreign_release_name.
    let ptr = unsafe { foreign_alloc_name(id) };
    if ptr.is_null() {
        return None;
    }
    let text = unsafe { CStr::from_ptr(ptr) }.to_string_lossy().into_owned();
    unsafe { foreign_release_name(ptr) };
    Some(text)
}
