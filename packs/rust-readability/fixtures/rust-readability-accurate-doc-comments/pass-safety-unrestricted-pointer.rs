/// Copies a sample from caller-supplied memory.
///
/// # Safety
/// sample must be non-null, aligned, and valid for reading an initialized u32.
/// Its allocation must remain live for the read, and the caller must ensure
/// that this non-atomic read does not race with a concurrent write.
pub unsafe fn copy_sample(sample: *const u32) -> u32 {
    unsafe { *sample }
}
