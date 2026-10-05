/// Split an internal scratch buffer at the boundary selected by its caller.
///
/// # Panics
/// Panics if boundary is beyond the end of the buffer. The caller must compute
/// the boundary from this buffer's layout before calling this internal helper.
pub fn split_scratch(buffer: &mut [u8], boundary: usize) -> (&mut [u8], &mut [u8]) {
    assert!(boundary <= buffer.len(), "scratch boundary is out of range");
    buffer.split_at_mut(boundary)
}
