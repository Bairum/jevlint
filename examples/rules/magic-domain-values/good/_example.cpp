const int kMaxRetryAttempts = 3;

bool can_retry(int attempts) {
	return attempts < kMaxRetryAttempts;
}
