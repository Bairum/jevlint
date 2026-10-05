/// Consume a working payload and return the replacement. Invalid lengths
/// return an error; the consumed working payload is not returned to the caller.
pub fn replace_owned(mut state: Vec<u8>, length: &str) -> Result<Vec<u8>, std::num::ParseIntError> {
    state.clear();
    let length = str::parse::<usize>(length)?;
    state.resize(length, 0);
    Ok(state)
}
