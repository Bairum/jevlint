function fetchPoints(userId: number): number {
	return userId * 2;
}

function getUserPoints(userId: number): number {
	if (userId <= 0) {
		return 0;
	}
	return fetchPoints(userId);
}
