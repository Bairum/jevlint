struct Server {
	host: String,
	port: u16,
}

impl Server {
	fn new(host: String, port: u16) -> Self {
		Server { host, port }
	}
}
