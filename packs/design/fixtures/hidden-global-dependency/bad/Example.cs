class TaxCalculator {
	private static int taxRate = 20;

	public int Calculate(int amount) {
		return amount * taxRate / 100;
	}
}
