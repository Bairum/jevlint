function computeTotal(quantity) {
	const listPrice = quantity * 12.5;
	const discounted = listPrice * 0.9;
	const total = discounted + 4.0;
	return total;
}
