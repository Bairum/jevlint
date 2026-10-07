static mut TAX_RATE: i32 = 20;

fn calculate_tax(amount: i32) -> i32 {
	unsafe { amount * TAX_RATE / 100 }
}
