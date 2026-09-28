int can_retry(int attempts) {
	if (attempts < 0) {
		return 0;
	}
	return attempts < 3;
}
