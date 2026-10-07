package bad

type AppOptions struct {
	EnableCache    bool
	EnableLogging  bool
	SendEmail      bool
	DarkMode       bool
	RetryCount     int
	DatabaseURL    string
	MaxConnections int
	SMTPHost       string
}
