const MAX_RETRY_ATTEMPTS = 3;

function canRetry(attempts) {
	return attempts < MAX_RETRY_ATTEMPTS;
}
