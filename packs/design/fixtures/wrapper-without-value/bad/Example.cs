class Example {
	int FetchPoints(int userId) {
		return userId * 2;
	}

	int GetUserPoints(int userId) {
		return FetchPoints(userId);
	}
}
