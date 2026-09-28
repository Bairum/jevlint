package bad

var globalTaxRate = 20

func CalculateTax(amount int) int {
	return amount * globalTaxRate / 100
}
