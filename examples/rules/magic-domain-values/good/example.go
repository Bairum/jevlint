package good

const MaxRetryAttempts = 3

func CanRetry(attempts int) bool {
	return attempts < MaxRetryAttempts
}
