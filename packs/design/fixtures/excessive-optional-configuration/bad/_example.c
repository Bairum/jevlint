struct AppOptions {
	bool enable_cache;
	bool enable_logging;
	bool send_email;
	bool dark_mode;
	int retry_count;
	char *database_url;
	int max_connections;
	char *smtp_host;
};
