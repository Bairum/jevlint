function applyTax(amount: number): number { return amount + amount * 0.08; }

function buildTotal(quantity: number, unitPrice: number): number {
	const subtotal = quantity * unitPrice;
	const discount = subtotal / 10;
	const taxed = applyTax(subtotal - discount);
	return taxed + 250;
}
