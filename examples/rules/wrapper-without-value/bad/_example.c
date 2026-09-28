int fetch_points(int userId) {
	return userId * 2;
}

int get_user_points(int userId) {
	return fetch_points(userId);
}
