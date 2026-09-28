class Example {
	int FetchPoints(int userId) {
		return userId * 2;
	}

	int GetUserPoints(int userId) {
		if (userId <= 0) {
			return 0;
		}
		return FetchPoints(userId);
	}
}
