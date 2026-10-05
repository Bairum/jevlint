use std::ffi::{c_char, CStr};

/// # Safety
/// ptr must be a non-null pointer to a foreign NUL-terminated string in one allocation.
/// Its initialized bytes through the terminator must remain live and immutable for 'a.
/// This function borrows the string; ownership and release responsibility stay with caller.
pub unsafe fn foreign_name<'a>(ptr: *const c_char) -> &'a CStr {
    unsafe { CStr::from_ptr(ptr) }
}

/// # Safety
/// ptr must be non-null and aligned for u32, including when count is zero.
/// It must address count initialized u32 values within one foreign allocation, with
/// count * size_of::<u32>() <= isize::MAX. The allocation must remain live and immutable
/// for 'a. The caller retains ownership and must not release it before the borrow ends.
pub unsafe fn foreign_words<'a>(ptr: *const u32, count: usize) -> &'a [u32] {
    unsafe { std::slice::from_raw_parts(ptr, count) }
}
