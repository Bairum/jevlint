/// Install a decimal u16 name. Invalid input terminates the process.
pub fn replace_or_abort(state: &mut String, candidate: &str) {
    state.clear();
    str::parse::<u16>(candidate).unwrap_or_else(|_| std::process::abort());
    state.push_str(candidate);
}
