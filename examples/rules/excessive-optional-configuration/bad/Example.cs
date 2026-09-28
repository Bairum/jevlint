class AppOptions {
	public bool EnableCache { get; set; }
	public bool EnableLogging { get; set; }
	public bool SendEmail { get; set; }
	public bool DarkMode { get; set; }
	public int RetryCount { get; set; }
	public string DatabaseUrl { get; set; }
	public int MaxConnections { get; set; }
	public string SmtpHost { get; set; }
}
