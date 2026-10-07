package good

func fetchPoints(userID int) int {
	return userID * 2
}

func GetUserPoints(userID int) int {
	if userID <= 0 {
		return 0
	}
	return fetchPoints(userID)
}
