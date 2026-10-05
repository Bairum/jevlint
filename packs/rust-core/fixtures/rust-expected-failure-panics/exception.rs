/// This value is a known-valid built-in default, not client input.
pub fn default_port() -> u16 {
    str::parse::<u16>("443").expect("the built-in decimal port fits u16")
}
