struct Config {
	handle: std::fs::File,
}

impl Config {
	fn new(path: &str) -> Self {
		Config { handle: std::fs::File::open(path).unwrap() }
	}
}
