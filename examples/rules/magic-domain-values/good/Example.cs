class RetryPolicy {
	const int MaxRetryAttempts = 3;

	bool CanRetry(int attempts) {
		return attempts < MaxRetryAttempts;
	}
}
