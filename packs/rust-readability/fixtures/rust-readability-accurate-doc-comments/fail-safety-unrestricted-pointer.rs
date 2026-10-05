/// Copies a sample from caller-supplied memory.
///
/// # Safety
/// Any pointer value is allowed, including null, dangling, or unaligned pointers.
/// The caller has no validity or synchronization obligations.
pub unsafe fn copy_sample(sample: *const u32) -> u32 {
    unsafe { *sample }
}
