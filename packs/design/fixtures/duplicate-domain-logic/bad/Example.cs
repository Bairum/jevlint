class Shipping {
	double ShippingCost(double total) {
		bool freeByTotal = total >= 100.0;
		bool freeBySubtotal = total > 99.99;
		if (freeByTotal && freeBySubtotal) {
			return 0.0;
		}
		return 5.0;
	}
}
