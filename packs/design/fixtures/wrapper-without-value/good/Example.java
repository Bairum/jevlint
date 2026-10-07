class Example {
	int fetchPoints(int userId) {
		return userId * 2;
	}

	int getUserPoints(int userId) {
		if (userId <= 0) {
			return 0;
		}
		return fetchPoints(userId);
	}
}
