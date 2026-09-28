class Loader {
	public void Load() {
		try {
			Connect();
		} catch (System.Exception error) {
			System.Console.WriteLine(error.Message);
		}
	}

	void Connect() {
	}
}
