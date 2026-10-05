use std::ffi::{c_char, CStr};

#[repr(C)]
pub struct Label {
    bytes: *const c_char,
    len: usize,
}

extern "C" {
    fn read_label(out: *mut Label) -> bool;
}

pub fn label_text() -> Option<String> {
    // On success read_label fills both fields with a non-null, static allocation of exactly
    // len initialized, nonzero bytes. No extra terminator byte is allocated.
    let mut label = Label { bytes: std::ptr::null(), len: 0 };
    if !unsafe { read_label(&mut label) } {
        return None;
    }
    Some(unsafe { CStr::from_ptr(label.bytes) }.to_string_lossy().into_owned())
}
