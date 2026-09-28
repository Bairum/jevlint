struct FormatOptions {
    uppercase: bool,
    padding: usize,
    verbose: bool,
}

fn format_name(name: &str, options: FormatOptions) -> String {
    name.trim().to_string()
}
