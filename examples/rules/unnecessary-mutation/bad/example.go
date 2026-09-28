package bad

func ComputeTotal(quantity float64) float64 {
	value := quantity
	value = value * 12.5
	value = value * 0.9
	value = value + 4.0
	return value
}
