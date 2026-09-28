class AppOptions:
    def __init__(self):
        self.enable_cache = False
        self.enable_logging = False
        self.send_email = False
        self.dark_mode = False
        self.retry_count = 0
        self.database_url = ""
        self.max_connections = 0
        self.smtp_host = ""
