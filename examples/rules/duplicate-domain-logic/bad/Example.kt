fun shippingCost(total: Double): Double {
	val freeByTotal = total >= 100.0
	val freeBySubtotal = total > 99.99
	if (freeByTotal && freeBySubtotal) {
		return 0.0
	}
	return 5.0
}
