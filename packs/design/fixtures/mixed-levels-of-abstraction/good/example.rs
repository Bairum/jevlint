fn calculate_subtotal(quantity: i32, unit_price: i32) -> i32 { quantity * unit_price }

fn apply_tax(amount: i32) -> i32 { amount + amount * 8 / 100 }

fn build_total(quantity: i32, unit_price: i32) -> i32 {
	let subtotal = calculate_subtotal(quantity, unit_price);
	apply_tax(subtotal)
}
