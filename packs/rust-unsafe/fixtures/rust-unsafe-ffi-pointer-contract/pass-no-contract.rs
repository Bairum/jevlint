use std::ffi::{c_char, CStr};

#[repr(C)]
pub struct Cursor {
    _private: [u8; 0],
}

extern "C" {
    fn cursor_name(cursor: *mut Cursor) -> *const c_char;
    fn cursor_advance(cursor: *mut Cursor);
}

pub unsafe fn next_name(cursor: *mut Cursor) -> String {
    let name = unsafe { cursor_name(cursor) };
    unsafe { cursor_advance(cursor) };
    unsafe { CStr::from_ptr(name) }.to_string_lossy().into_owned()
}
