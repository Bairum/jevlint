class Example {
	fun calculateSubtotal(quantity: Int, unitPrice: Int): Int = quantity * unitPrice

	fun applyTax(amount: Int): Int = amount + amount * 8 / 100

	fun buildTotal(quantity: Int, unitPrice: Int): Int {
		val subtotal = calculateSubtotal(quantity, unitPrice)
		return applyTax(subtotal)
	}
}
