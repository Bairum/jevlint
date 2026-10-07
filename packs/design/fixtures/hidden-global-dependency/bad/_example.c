int global_tax_rate = 20;

int calculate_tax(int amount) {
	return amount * global_tax_rate / 100;
}
