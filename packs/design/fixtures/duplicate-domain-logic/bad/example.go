package bad

func ShippingCost(total float64) float64 {
	freeByTotal := total >= 100.0
	freeBySubtotal := total > 99.99
	if freeByTotal && freeBySubtotal {
		return 0.0
	}
	return 5.0
}
