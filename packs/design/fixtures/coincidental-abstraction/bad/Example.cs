class SystemChecks {
	bool IsOverBudget(double amount) {
		return amount > 1000.0;
	}

	bool CheckSystems(double balance, double cpuLoad) {
		bool overBudget = IsOverBudget(balance);
		bool overloaded = IsOverBudget(cpuLoad);
		return overBudget || overloaded;
	}
}
