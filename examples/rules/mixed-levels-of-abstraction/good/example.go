package good

func calculateSubtotal(quantity int, unitPrice int) int { return quantity * unitPrice }

func applyTax(amount int) int { return amount + amount*8/100 }

func BuildTotal(quantity int, unitPrice int) int {
	subtotal := calculateSubtotal(quantity, unitPrice)
	return applyTax(subtotal)
}
