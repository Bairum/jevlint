package good

func ShippingCost(total float64) float64 {
	if total >= 100.0 {
		return 0.0
	}
	return 5.0
}
