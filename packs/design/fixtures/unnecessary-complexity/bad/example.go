package bad

func MaxValue(a int, b int) int {
	if a > b {
		return a
	} else {
		if b > a {
			return b
		} else {
			return a
		}
	}
}
