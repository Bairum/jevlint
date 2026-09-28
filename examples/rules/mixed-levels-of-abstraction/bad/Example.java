class Example {
	int applyTax(int amount) { return amount + amount * 8 / 100; }

	int buildTotal(int quantity, int unitPrice) {
		int subtotal = quantity * unitPrice;
		int discount = subtotal / 10;
		int taxed = applyTax(subtotal - discount);
		return taxed + 250;
	}
}
