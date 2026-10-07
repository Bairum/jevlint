package bad

func applyTax(amount int) int { return amount + amount*8/100 }

func BuildTotal(quantity int, unitPrice int) int {
	subtotal := quantity * unitPrice
	discount := subtotal / 10
	taxed := applyTax(subtotal - discount)
	return taxed + 250
}
