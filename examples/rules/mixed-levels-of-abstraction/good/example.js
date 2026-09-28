function calculateSubtotal(quantity, unitPrice) { return quantity * unitPrice; }

function applyTax(amount) { return amount + amount * 0.08; }

function buildTotal(quantity, unitPrice) {
	const subtotal = calculateSubtotal(quantity, unitPrice);
	return applyTax(subtotal);
}
