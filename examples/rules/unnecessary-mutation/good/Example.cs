class OrderCalculator {
	double ComputeTotal(double quantity) {
		double listPrice = quantity * 12.5;
		double discounted = listPrice * 0.9;
		double total = discounted + 4.0;
		return total;
	}
}
