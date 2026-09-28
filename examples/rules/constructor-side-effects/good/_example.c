struct Server {
	char *host;
	int port;
};

void server_create(struct Server *server, char *host, int port) {
	server->host = host;
	server->port = port;
}
