class BudgetCheck {
	boolean isOverBudget(double balance) {
		return balance > 1000.0;
	}
}

class LoadCheck {
	boolean isOverloaded(double cpuLoad) {
		return cpuLoad > 80.0;
	}
}
