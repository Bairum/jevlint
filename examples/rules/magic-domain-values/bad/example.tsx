function canRetry(attempts: number): boolean {
	if (attempts < 0) {
		return false;
	}
	return attempts < 3;
}
