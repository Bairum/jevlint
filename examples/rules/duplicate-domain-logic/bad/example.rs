fn shipping_cost(total: f64) -> f64 {
	let free_by_total = total >= 100.0;
	let free_by_subtotal = total > 99.99;
	if free_by_total && free_by_subtotal {
		0.0
	} else {
		5.0
	}
}
