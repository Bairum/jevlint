#![crate_type = "lib"]

use std::ffi::{c_char, c_void, CStr};

extern "C" {
    // For a valid string child and format "&s", the first variadic argument
    // must point to writable pointer storage. C writes a non-null pointer to
    // borrowed, readable NUL-terminated string data there. The string remains
    // valid while the immutable parent variant is alive and must not be freed.
    fn g_variant_get_child(value: *mut c_void, index: usize, format: *const c_char, ...);
}

/// # Safety
/// `variant` must point to a valid, immutable GLib GVariant of type `as`.
/// `index` must identify an existing child. The caller must keep the variant
/// alive and its borrowed string data readable and unchanged for all of `'a`.
pub unsafe fn variant_child_string<'a>(variant: *mut c_void, index: usize) -> &'a CStr {
    unsafe {
        let mut p: *mut c_char = std::ptr::null_mut();
        g_variant_get_child(
            variant,
            index,
            b"&s\0".as_ptr().cast::<c_char>(),
            &mut p,
            std::ptr::null::<c_char>(),
        );
        CStr::from_ptr(p)
    }
}
