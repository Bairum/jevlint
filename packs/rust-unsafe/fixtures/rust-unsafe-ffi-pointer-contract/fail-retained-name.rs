use std::ffi::{c_char, CStr};

#[repr(C)]
pub struct Cursor {
    _private: [u8; 0],
}

extern "C" {
    fn cursor_name(cursor: *mut Cursor) -> *const c_char;
    fn cursor_advance(cursor: *mut Cursor);
}

/// # Safety
/// cursor must designate a live, exclusively accessed foreign cursor.
pub unsafe fn next_name(cursor: *mut Cursor) -> String {
    // cursor_name returns a non-null NUL-terminated string valid until cursor_advance.
    // cursor_advance releases the current name allocation and loads the next record.
    let name = unsafe { cursor_name(cursor) };
    unsafe { cursor_advance(cursor) };
    unsafe { CStr::from_ptr(name) }.to_string_lossy().into_owned()
}
