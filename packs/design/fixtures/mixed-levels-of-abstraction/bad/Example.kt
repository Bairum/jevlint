class Example {
	fun applyTax(amount: Int): Int = amount + amount * 8 / 100

	fun buildTotal(quantity: Int, unitPrice: Int): Int {
		val subtotal = quantity * unitPrice
		val discount = subtotal / 10
		val taxed = applyTax(subtotal - discount)
		return taxed + 250
	}
}
