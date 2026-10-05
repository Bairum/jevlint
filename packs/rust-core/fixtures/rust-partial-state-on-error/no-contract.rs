pub fn replace_name(state: &mut String, candidate: &str) -> Result<(), std::num::ParseIntError> {
    state.clear();
    str::parse::<u16>(candidate)?;
    state.push_str(candidate);
    Ok(())
}
