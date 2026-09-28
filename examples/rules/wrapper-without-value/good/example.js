function fetchPoints(userId) {
	return userId * 2;
}

function getUserPoints(userId) {
	if (userId <= 0) {
		return 0;
	}
	return fetchPoints(userId);
}
