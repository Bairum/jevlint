function isOverBudget(balance: number): boolean {
	return balance > 1000;
}

function isOverloaded(cpuLoad: number): boolean {
	return cpuLoad > 80;
}
