function calculateSubtotal(quantity: number, unitPrice: number): number { return quantity * unitPrice; }

function applyTax(amount: number): number { return amount + amount * 0.08; }

function buildTotal(quantity: number, unitPrice: number): number {
	const subtotal = calculateSubtotal(quantity, unitPrice);
	return applyTax(subtotal);
}
