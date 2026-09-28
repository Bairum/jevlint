fun computeTotal(quantity: Double): Double {
	val listPrice = quantity * 12.5
	val discounted = listPrice * 0.9
	val total = discounted + 4.0
	return total
}
