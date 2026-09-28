class Config {
	private Object stream;

	Config(String path) {
		this.stream = new java.io.FileInputStream(path);
	}
}
