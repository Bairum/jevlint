#[derive(Debug)]
pub struct BlockError;

/// Called by the storage encoder after assigning fixed-size block slots.
///
/// # Panics
/// The encoder must supply an output slot of exactly eight bytes. A different
/// slot size is a programming error in the encoder, not a property of the record.
pub fn encode_record_number(text: &str, output_slot: &mut [u8]) -> Result<(), BlockError> {
    assert_eq!(output_slot.len(), 8, "encoder slot size");
    let number = text.parse::<u64>().map_err(|_| BlockError)?;
    output_slot.copy_from_slice(&number.to_le_bytes());
    Ok(())
}
