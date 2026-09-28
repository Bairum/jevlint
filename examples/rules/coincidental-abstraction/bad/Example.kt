class SystemChecks {
	fun isOverBudget(amount: Double): Boolean {
		return amount > 1000.0
	}

	fun checkSystems(balance: Double, cpuLoad: Double): Boolean {
		val overBudget = isOverBudget(balance)
		val overloaded = isOverBudget(cpuLoad)
		return overBudget || overloaded
	}
}
