class Example {
	int fetchPoints(int userId) {
		return userId * 2;
	}

	int getUserPoints(int userId) {
		return fetchPoints(userId);
	}
}
