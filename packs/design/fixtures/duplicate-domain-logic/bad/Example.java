class Shipping {
	double shippingCost(double total) {
		boolean freeByTotal = total >= 100.0;
		boolean freeBySubtotal = total > 99.99;
		if (freeByTotal && freeBySubtotal) {
			return 0.0;
		}
		return 5.0;
	}
}
