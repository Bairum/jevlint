package bad

func CanRetry(attempts int) bool {
	if attempts < 0 {
		return false
	}
	return attempts < 3
}
