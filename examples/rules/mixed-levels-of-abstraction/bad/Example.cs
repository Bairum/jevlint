class Example {
	int ApplyTax(int amount) { return amount + amount * 8 / 100; }

	int BuildTotal(int quantity, int unitPrice) {
		int subtotal = quantity * unitPrice;
		int discount = subtotal / 10;
		int taxed = ApplyTax(subtotal - discount);
		return taxed + 250;
	}
}
