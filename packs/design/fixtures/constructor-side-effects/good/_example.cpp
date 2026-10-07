class Server {
public:
	Server(const char *host, int port) : host(host), port(port) {}
private:
	const char *host;
	int port;
};
