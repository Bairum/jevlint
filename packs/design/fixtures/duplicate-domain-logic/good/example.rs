fn shipping_cost(total: f64) -> f64 {
	if total >= 100.0 {
		return 0.0;
	}
	5.0
}
