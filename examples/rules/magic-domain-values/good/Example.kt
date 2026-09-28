class RetryPolicy {
	companion object {
		const val MAX_RETRY_ATTEMPTS = 3
	}

	fun canRetry(attempts: Int): Boolean {
		return attempts < MAX_RETRY_ATTEMPTS
	}
}
