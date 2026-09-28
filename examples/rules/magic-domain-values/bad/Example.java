class RetryPolicy {
	boolean canRetry(int attempts) {
		return attempts < 3;
	}
}
