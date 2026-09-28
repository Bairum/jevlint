fn load() -> Result<(), String> {
	match connect() {
		Ok(()) => Ok(()),
		Err(error) => Err(error),
	}
}

fn connect() -> Result<(), String> {
	Ok(())
}
