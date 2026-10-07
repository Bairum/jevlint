int globalTaxRate = 20;

int CalculateTax(int amount) {
	return amount * globalTaxRate / 100;
}
