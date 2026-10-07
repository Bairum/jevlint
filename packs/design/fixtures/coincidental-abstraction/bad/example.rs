fn is_over_budget(amount: f64) -> bool {
	amount > 1000.0
}

fn check_systems(balance: f64, cpu_load: f64) -> bool {
	let over_budget = is_over_budget(balance);
	let overloaded = is_over_budget(cpu_load);
	over_budget || overloaded
}
