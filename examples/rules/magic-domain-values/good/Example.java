class RetryPolicy {
	static final int MAX_RETRY_ATTEMPTS = 3;

	boolean canRetry(int attempts) {
		return attempts < MAX_RETRY_ATTEMPTS;
	}
}
