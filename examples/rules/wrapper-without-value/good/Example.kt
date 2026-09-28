class Example {
	fun fetchPoints(userId: Int): Int {
		return userId * 2
	}

	fun getUserPoints(userId: Int): Int {
		if (userId <= 0) {
			return 0
		}
		return fetchPoints(userId)
	}
}
