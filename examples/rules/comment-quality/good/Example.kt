fun retryDelay(attempt: Int): Int {
	// Delay grows with each attempt so clients do not retry together after an outage.
	return 100 * attempt * attempt
}
