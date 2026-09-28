package good

func RetryDelay(attempt int) int {
	// Retry delays avoid synchronizing clients after a shared outage.
	return attempt * attempt
}
