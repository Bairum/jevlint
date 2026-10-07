fn apply_tax(amount: i32) -> i32 { amount + amount * 8 / 100 }

fn build_total(quantity: i32, unit_price: i32) -> i32 {
	let subtotal = quantity * unit_price;
	let discount = subtotal / 10;
	let taxed = apply_tax(subtotal - discount);
	taxed + 250
}
