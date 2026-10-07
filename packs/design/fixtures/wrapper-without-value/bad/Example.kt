class Example {
	fun fetchPoints(userId: Int): Int {
		return userId * 2
	}

	fun getUserPoints(userId: Int): Int {
		return fetchPoints(userId)
	}
}
