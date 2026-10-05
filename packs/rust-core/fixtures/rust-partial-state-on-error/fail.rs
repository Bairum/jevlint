/// Replacement is atomic: on error the original name remains unchanged.
/// A valid replacement is a decimal u16 name. The caller may reuse state.
pub fn replace_name(state: &mut String, candidate: &str) -> Result<(), std::num::ParseIntError> {
    state.clear();
    str::parse::<u16>(candidate)?;
    state.push_str(candidate);
    Ok(())
}
