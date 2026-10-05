use std::ffi::{c_char, CStr};

extern "C" {
    fn registry_name(id: u32) -> *const c_char;
}

pub fn unavailable_name(id: u32) -> Option<String> {
    // registry_name returns null for an absent entry, otherwise a static NUL-terminated string.
    let ptr = unsafe { registry_name(id) };
    if !ptr.is_null() {
        return None;
    }
    Some(unsafe { CStr::from_ptr(ptr) }.to_string_lossy().into_owned())
}
