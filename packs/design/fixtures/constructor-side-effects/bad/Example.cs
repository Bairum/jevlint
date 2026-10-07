class Config {
	string data;

	public Config(string path) {
		this.data = new System.IO.StreamReader(path).ReadToEnd();
	}
}
