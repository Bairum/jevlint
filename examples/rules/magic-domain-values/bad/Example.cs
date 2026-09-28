class RetryPolicy {
	bool CanRetry(int attempts) {
		return attempts < 3;
	}
}
