class Example {
	int CalculateSubtotal(int quantity, int unitPrice) { return quantity * unitPrice; }

	int ApplyTax(int amount) { return amount + amount * 8 / 100; }

	int BuildTotal(int quantity, int unitPrice) {
		int subtotal = CalculateSubtotal(quantity, unitPrice);
		return ApplyTax(subtotal);
	}
}
