struct AppOptions {
	bool enable_cache;
	bool enable_logging;
	bool send_email;
	bool dark_mode;
	int retry_count;
	const char *database_url;
	int max_connections;
	const char *smtp_host;
};
