package bad

func fetchPoints(userID int) int {
	return userID * 2
}

func GetUserPoints(userID int) int {
	return fetchPoints(userID)
}
