class Example {
	void load() {
		try {
			connect();
		} catch (Exception error) {
			throw new RuntimeException(error);
		}
	}

	void connect() {
	}
}
