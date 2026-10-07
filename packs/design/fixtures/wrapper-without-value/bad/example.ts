function fetchPoints(userId: number): number {
	return userId * 2;
}

function getUserPoints(userId: number): number {
	return fetchPoints(userId);
}
