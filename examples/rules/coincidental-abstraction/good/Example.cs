class BudgetCheck {
	bool IsOverBudget(double balance) {
		return balance > 1000.0;
	}
}

class LoadCheck {
	bool IsOverloaded(double cpuLoad) {
		return cpuLoad > 80.0;
	}
}
