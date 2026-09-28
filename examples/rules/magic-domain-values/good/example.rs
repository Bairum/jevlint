const MAX_RETRY_ATTEMPTS: i32 = 3;

fn can_retry(attempts: i32) -> bool {
	attempts < MAX_RETRY_ATTEMPTS
}
