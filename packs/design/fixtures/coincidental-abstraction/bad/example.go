package bad

func IsOverBudget(amount float64) bool {
	return amount > 1000.0
}

func CheckSystems(balance float64, cpuLoad float64) bool {
	overBudget := IsOverBudget(balance)
	overloaded := IsOverBudget(cpuLoad)
	return overBudget || overloaded
}
