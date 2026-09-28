class SystemChecks {
	boolean isOverBudget(double amount) {
		return amount > 1000.0;
	}

	boolean checkSystems(double balance, double cpuLoad) {
		boolean overBudget = isOverBudget(balance);
		boolean overloaded = isOverBudget(cpuLoad);
		return overBudget || overloaded;
	}
}
