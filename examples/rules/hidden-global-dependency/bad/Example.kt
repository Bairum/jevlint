class Example {
	companion object {
		var taxRate = 20
	}

	fun calculateTax(amount: Int): Int {
		return amount * taxRate / 100
	}
}
