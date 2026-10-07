struct AppOptions {
	enable_cache: bool,
	enable_logging: bool,
	send_email: bool,
	dark_mode: bool,
	retry_count: u32,
	database_url: String,
	max_connections: u32,
	smtp_host: String,
}
