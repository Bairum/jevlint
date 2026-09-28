class RetryPolicy {
	fun canRetry(attempts: Int): Boolean {
		return attempts < 3
	}
}
