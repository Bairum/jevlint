const MAX_RETRY_ATTEMPTS = 3;

function canRetry(attempts: number): boolean {
	return attempts < MAX_RETRY_ATTEMPTS;
}
