function shippingCost(total: number): number {
	const freeByTotal = total >= 100.0;
	const freeBySubtotal = total > 99.99;
	if (freeByTotal && freeBySubtotal) {
		return 0.0;
	}
	return 5.0;
}
