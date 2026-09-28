bool can_retry(int attempts) {
	if (attempts < 0) {
		return false;
	}
	return attempts < 3;
}
