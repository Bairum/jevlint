class BudgetCheck {
	fun isOverBudget(balance: Double): Boolean {
		return balance > 1000.0
	}
}

class LoadCheck {
	fun isOverloaded(cpuLoad: Double): Boolean {
		return cpuLoad > 80.0
	}
}
