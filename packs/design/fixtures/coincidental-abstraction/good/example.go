package good

func IsOverBudget(balance float64) bool {
	return balance > 1000.0
}

func IsOverloaded(cpuLoad float64) bool {
	return cpuLoad > 80.0
}
