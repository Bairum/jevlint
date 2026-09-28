function applyTax(amount) { return amount + amount * 0.08; }

function buildTotal(quantity, unitPrice) {
	const subtotal = quantity * unitPrice;
	const discount = subtotal / 10;
	const taxed = applyTax(subtotal - discount);
	return taxed + 250;
}
