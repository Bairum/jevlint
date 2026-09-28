fn retry_delay(attempt: i32) -> i32 {
	// Delay grows with each attempt so clients do not retry together after an outage.
	100 * attempt * attempt
}
