package bad

func CountPositivePairs(values []int) int {
	pairs := 0
	for i, first := range values {
		if first > 0 {
			for _, second := range values[i+1:] {
				if second > 0 {
					pairs++
				}
			}
		}
	}
	return pairs
}
