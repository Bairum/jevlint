fn compute_total(quantity: f64) -> f64 {
	let list_price = quantity * 12.5;
	let discounted = list_price * 0.9;
	let total = discounted + 4.0;
	total
}
