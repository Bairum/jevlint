function retryDelay(attempt) {
	// Delay grows with each attempt so clients do not retry together after an outage.
	return 100 * attempt * attempt;
}
