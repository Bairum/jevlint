int fetch_points(int userId) {
	return userId * 2;
}

int get_user_points(int userId) {
	if (userId <= 0) {
		return 0;
	}
	return fetch_points(userId);
}
