class AppOptions
  def initialize
    @enable_cache = false
    @enable_logging = false
    @send_email = false
    @dark_mode = false
    @retry_count = 0
    @database_url = ""
    @max_connections = 0
    @smtp_host = ""
  end
end
