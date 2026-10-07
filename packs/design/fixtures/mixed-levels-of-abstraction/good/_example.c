int calculateSubtotal(int quantity, int unitPrice) { return quantity * unitPrice; }

int applyTax(int amount) { return amount + amount * 8 / 100; }

int buildTotal(int quantity, int unitPrice) {
	int subtotal = calculateSubtotal(quantity, unitPrice);
	return applyTax(subtotal);
}
