fn can_retry(attempts: i32) -> bool {
	if attempts < 0 {
		return false;
	}
	attempts < 3
}
