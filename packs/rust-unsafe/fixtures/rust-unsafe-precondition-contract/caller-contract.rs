/// # Safety
/// `pointer` must address `length` initialized u32 values in one live allocation
/// for the returned lifetime, with no mutation while the slice is borrowed.
pub unsafe fn view<'a>(pointer: *const u32, length: usize) -> &'a [u32] {
    unsafe { std::slice::from_raw_parts(pointer, length) }
}
