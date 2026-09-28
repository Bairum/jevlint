#define MAX_RETRY_ATTEMPTS 3

int can_retry(int attempts) {
	return attempts < MAX_RETRY_ATTEMPTS;
}
