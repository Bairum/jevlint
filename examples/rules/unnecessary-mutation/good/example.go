package good

func ComputeTotal(quantity float64) float64 {
	listPrice := quantity * 12.5
	discounted := listPrice * 0.9
	total := discounted + 4.0
	return total
}
