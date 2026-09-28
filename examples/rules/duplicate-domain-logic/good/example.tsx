function shippingCost(total: number): number {
	if (total >= 100.0) {
		return 0.0;
	}
	return 5.0;
}
